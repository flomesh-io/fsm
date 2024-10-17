((
  config = pipy.solve('config.js'),
  specServiceIdentity = config?.Spec?.ServiceIdentity,
  {
    shuffle,
    failover,
  } = pipy.solve('utils.js'),

  allMethods = ['GET', 'HEAD', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS'],

  clusterCache = new algo.Cache(
    (clusterName => (
      (cluster = config?.Outbound?.ClustersConfigs?.[clusterName]) => (
        cluster ? Object.assign({ name: clusterName }, cluster) : null
      )
    )())
  ),

  makeServiceHandler = (portConfig, serviceInfo) => (
    (
      rules = portConfig?.HttpServiceRouteRules?.[serviceInfo?.RuleName]?.RouteRules || [],
      tree = {},
    ) => (
      rules.forEach(
        config => (
          (
            matchPath = (
              (config.Type === 'Regex') && (
                ((match = null) => (
                  match = new RegExp(config.Path),
                  (path) => match.test(path)
                ))()
              ) || (config.Type === 'Exact') && (
                (path) => path === config.Path
              ) || (config.Type === 'Prefix') && (
                (path) => path.startsWith(config.Path)
              )
            ),
            headerRules = config.Headers ? Object.entries(config.Headers).map(([k, v]) => [k, new RegExp(v)]) : null,
            balancer = new algo.RoundRobinLoadBalancer(shuffle(config.TargetClusters || {})),
            failoverBalancer = failover(config.TargetClusters),
            service = Object.assign({ name: serviceInfo.Service || serviceInfo?.RuleName }, portConfig?.HttpServiceRouteRules?.[serviceInfo.RuleName]),
            rule = headerRules ? (
              (path, headers) => matchPath(path) && headerRules.every(([k, v]) => v.test(headers[k] || '')) && (
                __route = config,
                __service = service,
                __cluster = clusterCache.get(balancer.borrow({})?.id),
                failoverBalancer && (
                  _failoverCluster = clusterCache.get(failoverBalancer.borrow()?.id)
                ),
                true
              )
            ) : (
              (path) => matchPath(path) && (
                __route = config,
                __service = service,
                __cluster = clusterCache.get(balancer.borrow({})?.id),
                failoverBalancer && (
                  _failoverCluster = clusterCache.get(failoverBalancer.borrow()?.id)
                ),
                true
              )
            ),
            allowedMethods = config.Methods || allMethods,
          ) => (
            allowedMethods.forEach(
              method => (tree[method] || (tree[method] = [])).push(rule)
            )
          )
        )()
      ),

      (method, path, headers) => void (
        tree[method]?.find?.(rule => rule(path, headers)),
        __service && (
          headers['serviceidentity'] = specServiceIdentity
        )
      )
    )
  )(),

  makePortHandler = (portConfig) => (
    (
      serviceHandlers = new algo.Cache(
        (serviceInfo) => makeServiceHandler(portConfig, serviceInfo)
      ),

      hostHandlers = new algo.Cache(
        (host) => (
          (
            vh = portConfig?.HttpHostPort2Service?.[host],
            newHost,
          ) => (
            !vh && config?.Spec?.FeatureFlags?.EnableAutoDefaultRoute && (
              vh = portConfig?.HttpHostPort2Service?.[Object.keys(portConfig.HttpHostPort2Service)[0]],
              newHost = vh.Service || vh.RuleName
            ),
            { handler: serviceHandlers.get(vh), newHost }
          )
        )()
      ),
    ) => (
      (msg) => (
        (
          head = msg.head,
          headers = head.headers,
          svcStruct = hostHandlers.get(headers.host),
        ) => (
          svcStruct.handler && (
            svcStruct.newHost && (
              headers.host = svcStruct.newHost
            ),
            svcStruct.handler(head.method, head.path, headers)
          )
        )
      )()
    )
  )(),

  portHandlers = new algo.Cache(makePortHandler),
) => pipy({
  _origPath: null,
  _failoverCluster: null,
  _useHttp2: false,
})

.import({
  __port: 'outbound',
  __protocol: 'outbound',
  __isHTTP2: 'outbound',
})

.export('outbound-http-routing', {
  __route: null,
  __service: null,
  __cluster: null,
})

.pipeline()
.branch(
  () => (__protocol === 'http') && !__isHTTP2, (
    $=>$.detectProtocol(
      proto => proto === 'HTTP2' && (_useHttp2 = true)
    )
  ), (
    $=>$
  )
)
.demuxHTTP().to(
  $=>$
  .replay({ 'delay': 0 }).to(
    $=>$
    .handleMessageStart(
      msg => (
        _useHttp2 && msg?.head?.headers?.['content-type'] === 'application/grpc' && (
          __isHTTP2 = true
        ),
        _origPath && (msg.head.path = _origPath) || (_origPath = msg?.head?.path),
        _failoverCluster && (
          __cluster = _failoverCluster,
          _failoverCluster = null,
          true
        ) || (
          portHandlers.get(__port)(msg)
        )
      )
    )
    .chain()
    .replaceMessage(
      msg => (
        (
          status = msg?.head?.status
        ) => (
          _failoverCluster && (!status || status > '499') ? new StreamEnd('Replay') : msg
        )
      )()
    )
  )
)

)()
