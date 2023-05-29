((
  config = pipy.solve('config.js'),
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

  makeServiceHandler = (portConfig, serviceName) => (
    (
      rules = portConfig?.HttpServiceRouteRules?.[serviceName]?.RouteRules || [],
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
            service = Object.assign({ name: serviceName }, portConfig?.HttpServiceRouteRules?.[serviceName]),
            rule = headerRules ? (
              (path, headers) => matchPath(path) && headerRules.every(([k, v]) => v.test(headers[k] || '')) && (
                __route = config,
                __service = service,
                __cluster = clusterCache.get(balancer.next({})?.id),
                failoverBalancer && (
                  _failoverCluster = clusterCache.get(failoverBalancer.next({})?.id)
                ),
                true
              )
            ) : (
              (path) => matchPath(path) && (
                __route = config,
                __service = service,
                __cluster = clusterCache.get(balancer.next({})?.id),
                failoverBalancer && (
                  _failoverCluster = clusterCache.get(failoverBalancer.next({})?.id)
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
          headers['serviceidentity'] = __service.ServiceIdentity
        )
      )
    )
  )(),

  makePortHandler = (portConfig) => (
    (
      serviceHandlers = new algo.Cache(
        (serviceName) => makeServiceHandler(portConfig, serviceName)
      ),

      hostHandlers = new algo.Cache(
        (host) => serviceHandlers.get(portConfig?.HttpHostPort2Service?.[host])
      ),
    ) => (
      (msg) => (
        (
          head = msg.head,
          headers = head.headers,
        ) => (
          hostHandlers.get(headers.host)(head.method, head.path, headers)
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
