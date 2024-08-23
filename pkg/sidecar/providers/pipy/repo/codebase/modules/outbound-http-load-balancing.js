((
  config = pipy.solve('config.js'),
  http1PerRequestLoadBalancing = Boolean(config?.Spec?.Traffic?.HTTP1PerRequestLoadBalancing),
  http2PerRequestLoadBalancing = Boolean(config?.Spec?.Traffic?.HTTP2PerRequestLoadBalancing),
  certChain = config?.Certificate?.CertChain,
  privateKey = config?.Certificate?.PrivateKey,
  isDebugEnabled = config?.Spec?.SidecarLogLevel === 'debug',
  {
    shuffle,
    failover,
  } = pipy.solve('utils.js'),

  retryCounter = new stats.Counter('sidecar_cluster_upstream_rq_retry', ['sidecar_cluster_name']),
  retrySuccessCounter = new stats.Counter('sidecar_cluster_upstream_rq_retry_success', ['sidecar_cluster_name']),
  retryLimitCounter = new stats.Counter('sidecar_cluster_upstream_rq_retry_limit_exceeded', ['sidecar_cluster_name']),
  retryOverflowCounter = new stats.Counter('sidecar_cluster_upstream_rq_retry_overflow', ['sidecar_cluster_name']),
  retryBackoffCounter = new stats.Counter('sidecar_cluster_upstream_rq_retry_backoff_exponential', ['sidecar_cluster_name']),
  retryBackoffLimitCounter = new stats.Counter('sidecar_cluster_upstream_rq_retry_backoff_ratelimited', ['sidecar_cluster_name']),

  makeClusterConfig = (clusterConfig) => (
    clusterConfig && (
      (
        endpointAttributes = {},
        obj = {
          targetBalancer: clusterConfig.Endpoints && new algo.RoundRobinLoadBalancer(
            shuffle(Object.fromEntries(Object.entries(clusterConfig.Endpoints).map(([k, v]) => (endpointAttributes[k] = v, [k, v.Weight]))))
          ),
          endpointAttributes,
          failoverBalancer: clusterConfig.Endpoints && failover(Object.fromEntries(Object.entries(clusterConfig.Endpoints).map(([k, v]) => [k, v.Weight]))),
          needRetry: Boolean(clusterConfig.RetryPolicy?.NumRetries),
          numRetries: clusterConfig.RetryPolicy?.NumRetries,
          retryStatusCodes: (clusterConfig.RetryPolicy?.RetryOn || '5xx').split(',').reduce(
            (lut, code) => (
              code.endsWith('xx') ? (
                new Array(100).fill(0).forEach((_, i) => lut[(code.charAt(0)|0)*100+i] = true)
              ) : (
                lut[code|0] = true
              ),
              lut
            ),
            []
          ),
          retryBackoffBaseInterval: clusterConfig.RetryPolicy?.RetryBackoffBaseInterval > 1 ? 1 : clusterConfig.RetryPolicy?.RetryBackoffBaseInterval,
          retryCounter: retryCounter.withLabels(clusterConfig.name),
          retrySuccessCounter: retrySuccessCounter.withLabels(clusterConfig.name),
          retryLimitCounter: retryLimitCounter.withLabels(clusterConfig.name),
          retryOverflowCounter: retryOverflowCounter.withLabels(clusterConfig.name),
          retryBackoffCounter: retryBackoffCounter.withLabels(clusterConfig.name),
          retryBackoffLimitCounter: retryBackoffLimitCounter.withLabels(clusterConfig.name),
          muxHttpOptions: {
            version: () => __isHTTP2 ? 2 : 1,
            maxMessages: clusterConfig.ConnectionSettings?.http?.MaxRequestsPerConnection
          },
        },
      ) => (
        obj.retryCounter.zero(),
        obj.retrySuccessCounter.zero(),
        obj.retryLimitCounter.zero(),
        obj.retryOverflowCounter.zero(),
        obj.retryBackoffCounter.zero(),
        obj.retryBackoffLimitCounter.zero(),
        obj
      )
    )()
  ),

  clusterConfigs = new algo.Cache(makeClusterConfig),

  shouldRetry = (statusCode) => (
    _clusterConfig.retryStatusCodes[statusCode] ? (
      (_retryCount < _clusterConfig.numRetries) ? (
        _clusterConfig.retryCounter.increase(),
        _clusterConfig.retryBackoffCounter.increase(),
        _retryCount++,
        true
      ) : (
        _clusterConfig.retryLimitCounter.increase(),
        false
      )
    ) : (
      _retryCount > 0 && _clusterConfig.retrySuccessCounter.increase(),
      false
    )
  ),
) => pipy({
  _retryCount: 0,
  _clusterConfig: null,
  _failoverObject: null,
  _targetObject: null,
  _muxHttpOptions: null,
  _session: null,
})

.import({
  __port: 'outbound',
  __cert: 'outbound',
  __isHTTP2: 'outbound',
  __isEgress: 'outbound',
  __route: 'outbound-http-routing',
  __service: 'outbound-http-routing',
  __cluster: 'outbound-http-routing',
  __metricLabel: 'connect-tcp',
  __target: 'connect-tcp',
})

.pipeline()
.onStart(
  () => void (
    _session = {},
    (_clusterConfig = clusterConfigs.get(__cluster)) && (
      _muxHttpOptions = _clusterConfig.muxHttpOptions,
      _clusterConfig.failoverBalancer && (
        _failoverObject = _clusterConfig.failoverBalancer.borrow()
      )
    )
  )
)
.onEnd(() => void ( _session = null))
.handleMessageStart(
  msg => (
    _clusterConfig && (
      __isHTTP2 ? (
        http2PerRequestLoadBalancing ? (
          _targetObject = _clusterConfig.targetBalancer?.borrow?.({})
        ) : (
          _targetObject = _clusterConfig.targetBalancer?.borrow?.()
        )
      ) : (
        http1PerRequestLoadBalancing ? (
          _targetObject = _clusterConfig.targetBalancer?.borrow?.(_session)
        ) : (
          _targetObject = _clusterConfig.targetBalancer?.borrow?.()
        )
      ),
      __target = _targetObject?.id
    ) && (
      (
        attrs = _clusterConfig?.endpointAttributes?.[__target]
      ) => (
        attrs?.ViaGateway && (
          __isEgress = true,
          msg.head.headers['fgw-forwarded-host'] = msg.head.headers.host,
          msg.head.headers['fgw-forwarded-target'] = __target,
          msg.head.headers['fgw-forwarded-service'] = __service.name.split('.')[0],
          __target = attrs.ViaGateway
        ),
        attrs?.Path && (
          __isEgress = true,
          msg.head.path = attrs.Path + msg.head.path
        )
      )
    )()
  )
)

.branch(
  () => _clusterConfig?.needRetry, (
    $=>$
    .replay({
        delay: () => _clusterConfig.retryBackoffBaseInterval * Math.min(10, Math.pow(2, _retryCount-1)|0)
    }).to(
      $=>$
      .link('upstream')
      .replaceMessageStart(
        msg => (
          shouldRetry(msg.head.status) ? new StreamEnd('Replay') : msg
        )
      )
    )
  ),

  () => _failoverObject, (
    $=>$
    .replay({ 'delay': 0 }).to(
      $=>$
      .link('upstream')
      .replaceMessage(
        msg => (
          (
            status = msg?.head?.status
          ) => (
            _failoverObject && (!status || status > '499') ? (
              _targetObject = _failoverObject,
              __target = _targetObject.id,
              _failoverObject = null,
              new StreamEnd('Replay')
            ) : msg
          )
        )()
      )
    )
  ),

  (
    $=>$.link('upstream')
  )
)

.pipeline('upstream')
.handleStreamStart(
  () => (
    !__cert && __cluster?.SourceCert && (
      __cluster.SourceCert.FsmIssued && (
        __cert = {CertChain: certChain, PrivateKey: privateKey}
      ) || (
        __cert = __cluster.SourceCert
      )
    ),
    __metricLabel = __cluster?.name
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('outbound-http # port/service/route/cluster/egress/cert :',
          __port?.Port, __service?.name, __route?.Path, __cluster?.name, __isEgress, Boolean(__cert))
      )
    )
  )
)
.branch(
  () => !__target, (
    $=>$.chain()
  ),
  (
    $=>$.muxHTTP(() => _targetObject, () => _muxHttpOptions).to($=>$.use('connect-upstream.js'))
  )
)

)()
