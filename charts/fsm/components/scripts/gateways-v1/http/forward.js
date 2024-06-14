((
  { config, isDebugEnabled } = pipy.solve('config.js'),

  { healthCheckTargets, healthCheckServices } = pipy.solve('common/variables.js'),

  {
    shuffle,
    failover,
  } = pipy.solve('lib/utils.js'),

  http1PerRequestLoadBalancing = Boolean(config?.Configs?.HTTP1PerRequestLoadBalancing),
  http2PerRequestLoadBalancing = (config?.Configs?.HTTP2PerRequestLoadBalancing === undefined) || Boolean(config?.Configs?.HTTP2PerRequestLoadBalancing),

  retryCounter = new stats.Counter('fgw_upstream_rq_retry', ['service_name']),
  retrySuccessCounter = new stats.Counter('fgw_upstream_rq_retry_success', ['service_name']),
  retryLimitCounter = new stats.Counter('fgw_upstream_rq_retry_limit_exceeded', ['service_name']),
  retryOverflowCounter = new stats.Counter('fgw_upstream_rq_retry_overflow', ['service_name']),
  retryBackoffCounter = new stats.Counter('fgw_upstream_rq_retry_backoff_exponential', ['service_name']),
  retryBackoffLimitCounter = new stats.Counter('fgw_upstream_rq_retry_backoff_ratelimited', ['service_name']),

  makeServiceConfig = (serviceConfig) => (
    serviceConfig && (
      (
        endpointAttributes = {},
        endpoints = shuffle(
          Object.fromEntries(
            Object.entries(serviceConfig.Endpoints)
              .map(([k, v]) => (endpointAttributes[k] = v, v.hash = algo.hash(k), [k, v.Weight]))
              .filter(([k, v]) => (serviceConfig.Algorithm !== 'RoundRobinLoadBalancer' || v > 0))
          )
        ),
        obj = {
          targetBalancer: serviceConfig.Endpoints && (
            (serviceConfig.Algorithm === 'HashingLoadBalancer') ? (
              new algo.HashingLoadBalancer(Object.keys(endpoints))
            ) : (
              (serviceConfig.Algorithm === 'LeastConnectionLoadBalancer') ? (
                new algo.LeastWorkLoadBalancer(Object.keys(endpoints))
              ) : (
                new algo[serviceConfig.Algorithm || 'RoundRobinLoadBalancer'](endpoints)
              )
            )
          ),
          endpointAttributes,
          ...(serviceConfig.StickyCookieName && ({
            stickyCookie: {
              name: serviceConfig.StickyCookieName,
              expires: serviceConfig.StickyCookieExpires || 3600,
              hashTable: Object.fromEntries(Object.keys(serviceConfig.Endpoints).map(
                k => (
                  [algo.hash(k), k]
                )
              ))
            }
          })),
          failoverBalancer: serviceConfig.Endpoints && failover(Object.fromEntries(Object.entries(serviceConfig.Endpoints).map(([k, v]) => [k, v.Weight]))),
          needRetry: Boolean(serviceConfig.Retry?.NumRetries),
          numRetries: serviceConfig.Retry?.NumRetries,
          retryStatusCodes: (serviceConfig.Retry?.RetryOn || '5xx').split(',').reduce(
            (lut, code) => (
              code.endsWith('xx') ? (
                new Array(100).fill(0).forEach((_, i) => lut[(code.charAt(0) | 0) * 100 + i] = true)
              ) : (
                lut[code | 0] = true
              ),
              lut
            ),
            []
          ),
          retryBackoffBaseInterval: serviceConfig.Retry?.BackoffBaseInterval > 1 ? 1 : serviceConfig.Retry?.BackoffBaseInterval,
          retryCounter: retryCounter.withLabels(serviceConfig.name),
          retrySuccessCounter: retrySuccessCounter.withLabels(serviceConfig.name),
          retryLimitCounter: retryLimitCounter.withLabels(serviceConfig.name),
          retryOverflowCounter: retryOverflowCounter.withLabels(serviceConfig.name),
          retryBackoffCounter: retryBackoffCounter.withLabels(serviceConfig.name),
          retryBackoffLimitCounter: retryBackoffLimitCounter.withLabels(serviceConfig.name),
          muxHttpOptions: {
            version: () => (__domain?.RouteType === 'GRPC' || __domain?.RouteType === 'HTTP2') ? 2 : 1,
            maxMessages: serviceConfig.MaxRequestsPerConnection
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

  serviceConfigs = new algo.Cache(makeServiceConfig),

  shouldRetry = (statusCode) => (
    (
      again = _serviceConfig?.retryStatusCodes?.[statusCode] ? (
        (_retryCount < _serviceConfig.numRetries) ? (
          _serviceConfig.retryCounter.increase(),
          _serviceConfig.retryBackoffCounter.increase(),
          _retryCount++,
          true
        ) : (
          _serviceConfig.retryLimitCounter.increase(),
          false
        )
      ) : (
        _retryCount > 0 && _serviceConfig.retrySuccessCounter.increase(),
        false
      )
    ) => (
      (!again && _failoverBalancer && (!statusCode || statusCode > '499')) ? (
        _targetBalancer = _failoverBalancer,
        _failoverBalancer = null,
        _retryCount = 0,
        true
      ) : again
    )
  )(),

  getCookies = cookie => (
    (
      cookies = {},
      arr,
      kv,
    ) => (
      cookie && (
        (arr = cookie.split(';')) && (
          arr.forEach(
            p => (
              kv = p.split('='),
              (kv.length > 1) && (
                cookies[kv[0].trim()] = kv[1].trim()
              )
            )
          )
        ),
        cookies
      )
    )
  )(),

  proxyPreserveHostCache = new algo.Cache(
    route => !(route?.config?.ProxyPreserveHost === false || __domain?.ProxyPreserveHost === false || config?.Configs?.ProxyPreserveHost === false)
  ),

) => pipy({
  _retryCount: 0,
  _serviceConfig: null,
  _targetBalancer: null,
  _failoverBalancer: null,
  _muxHttpOptions: null,
  _version: null,
  _session: null,
  _cookies: null,
  _cookieId: null,
  _isRetry: false,
  _unhealthCache: null,
  _healthCheckTarget: null,
  _targetResource: null,
  _balancerKey: undefined,
})

.import({
  __domain: 'route',
  __route: 'route',
  __service: 'service',
  __cert: 'connect-tls',
  __host: 'connect-tls',
  __useSSL: 'connect-tls',
  __target: 'connect-tcp',
  __metricLabel: 'connect-tcp',
  __upstreamError: 'connect-tcp',
  __responseHead: 'http',
  __responseTail: 'http',
})

.pipeline()
.onStart(
  () => void (
    (_serviceConfig = serviceConfigs.get(__service)) && (
      __metricLabel = __service.name,
      _muxHttpOptions = _serviceConfig.muxHttpOptions,
      _version = _muxHttpOptions.version(),
      _session = {},
      _targetBalancer = _serviceConfig.targetBalancer,
      _serviceConfig.failoverBalancer && (
        _failoverBalancer = _serviceConfig.failoverBalancer
      ),
      _unhealthCache = healthCheckServices?.[__service.name]
    )
  )
)
.onEnd(() => void (_session = null))
.branch(
  () => _serviceConfig?.needRetry || _failoverBalancer, (
    $=>$
    .replay({
      delay: () => _serviceConfig.retryBackoffBaseInterval * Math.min(10, Math.pow(2, _retryCount - 1) | 0)
    }).to(
      $=>$
      .link('upstream')
      .replaceMessageStart(
        msg => (
          shouldRetry(msg?.head?.status) ? (
            _isRetry = true,
            new StreamEnd('Replay')
          ) : msg
        )
      )
    )
  ), (
    $=>$.link('upstream')
  )
)

.pipeline('upstream')
.handleMessageStart(
  msg => (
    _serviceConfig && (
      _serviceConfig.stickyCookie && (
        _cookieId = null,
        !_isRetry && (_cookies = getCookies(msg?.head?.headers?.cookie)) && (_cookieId = _cookies[_serviceConfig.stickyCookie.name]) && (
          (_cookieId = _serviceConfig.stickyCookie.hashTable[_cookieId]) && (
            _unhealthCache?.get?.(_cookieId) && (_cookieId = null)
          )
        )
      ),
      _cookieId ? (
        __target = _cookieId
      ) : (
        (__service?.Algorithm === 'HashingLoadBalancer') && (_balancerKey = __inbound.remoteAddress),
        (_version === 2) ? (
          http2PerRequestLoadBalancing ? (
            _targetResource = _targetBalancer?.borrow?.({}, _balancerKey, _unhealthCache)
          ) : (
            _targetResource = _targetBalancer?.borrow?.(__inbound, _balancerKey, _unhealthCache)
          )
        ) : http1PerRequestLoadBalancing ? (
          _targetResource = _targetBalancer?.borrow?.(_session, _balancerKey, _unhealthCache)
        ) : (
          _targetResource = _targetBalancer?.borrow?.(__inbound, _balancerKey, _unhealthCache)
        ),
        _targetResource && (
          __target = _targetResource?.id
        )
      ),
      __target
    ) && (
      (
        attrs = _serviceConfig?.endpointAttributes?.[__target]
      ) => (
        (__host = attrs?.Host) ? (
          msg.head.headers.host = __host
        ) : !proxyPreserveHostCache.get(__route) && (
          msg.head.headers.host = __target
        ),
        attrs?.UpstreamCert ? (
          __cert = attrs?.UpstreamCert
        ) : (
          __cert = __service?.UpstreamCert
        ),
        __cert ? (
          !__service?.MTLS && (
            __useSSL = true,
            __cert = null
          )
        ) : (
          __useSSL = Boolean(attrs?.UseSSL)
        ),
        _cookieId ? (
          _cookieId = null
        ) : (
          _serviceConfig?.stickyCookie && attrs?.hash && (
            _cookieId = _serviceConfig.stickyCookie.name + '=' + attrs.hash + '; path=/; expires='
                      + new Date(new Date().getTime() + 1000 * _serviceConfig.stickyCookie.expires).toUTCString()
                      + '; max-age=' + _serviceConfig.stickyCookie.expires
          )
        )
      )
    )()
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('[forward] target, cert, host/sni, useSSL:', __target, Boolean(__cert), __host, __useSSL)
      )
    )
  )
)
.branch(
  () => !__target, (
    $=>$
    .branch(
      () => _failoverBalancer, (
        $=>$.replaceMessageStart(
          () => (
            _targetBalancer = _failoverBalancer,
            _failoverBalancer = null,
            _isRetry = true,
            new StreamEnd('Replay')
          )
        )
      ), (
        $=>$.chain()
      )
    )
  ),
  (
    $=>$.muxHTTP(() => _targetResource, () => _muxHttpOptions).to(
      $=>$.branch(
        () => __cert || __useSSL, (
          $=>$.use('lib/connect-tls.js')
        ), (
          $=>$.use('lib/connect-tcp.js')
        )
      )
    )
    .handleMessage(
      msg => (
        config?.Configs?.ShowUpstreamStatusInResponseHeader && (
          (msg?.head?.status > 399) && (__upstreamError != 'ConnectionRefused') && (
            msg.head.headers ? (
              msg.head.headers['X-FGW-Upstream-Status'] = msg.head.status
            ) : (
              msg.head.headers = { 'X-FGW-Upstream-Status': msg.head.status }
            )
          )
        ),
        (_healthCheckTarget = healthCheckTargets?.[__target + '@' + __service.name]) && (
          (__upstreamError === 'ConnectionRefused') && (
            _healthCheckTarget.service.fail(_healthCheckTarget),
            _healthCheckTarget.reason = 'ConnectionRefused'
          )
        )
      )
    )
    .branch(
      () => _cookieId, (
        $=>$
        .handleMessageStart(
          msg => (
            __responseHead = msg.head,
            msg?.head?.headers && (
              !msg.head.headers['set-cookie'] && (
                msg.head.headers['set-cookie'] = []
              ),
              (typeof msg.head.headers['set-cookie'] === 'string') ? (
                msg.head.headers['set-cookie'] = [msg.head.headers['set-cookie'], _cookieId]
              ) : (
                msg.head.headers['set-cookie'].push(_cookieId)
              )
            )
          )
        )
        .handleMessageEnd(
          msg => __responseTail = msg.tail
        )
      ), (
        $=>$
      )
    )
  )
)

)()