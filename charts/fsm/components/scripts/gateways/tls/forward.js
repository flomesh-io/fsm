((
  { config, isDebugEnabled } = pipy.solve('config.js'),

  {
    shuffle,
    failover,
  } = pipy.solve('lib/utils.js'),

  makeServiceHandler = serviceName => (
    config?.Services?.[serviceName] ? (
      config.Services[serviceName].name = serviceName,
      config.Services[serviceName]
    ) : null
  ),

  serviceHandlers = new algo.Cache(makeServiceHandler),

  makeServiceConfig = (serviceConfig) => (
    serviceConfig && (
      (
        endpointAttributes = {},
        obj = {
          targetBalancer: serviceConfig.Endpoints && new algo.RoundRobinLoadBalancer(
            shuffle(Object.fromEntries(Object.entries(serviceConfig.Endpoints)
              .map(([k, v]) => (endpointAttributes[k] = v, [k, v.Weight]))
              .filter(([k, v]) => v > 0)
            ))
          ),
          endpointAttributes,
          failoverBalancer: serviceConfig.Endpoints && failover(Object.fromEntries(Object.entries(serviceConfig.Endpoints).map(([k, v]) => [k, v.Weight]))),
        },
      ) => (
        obj
      )
    )()
  ),

  serviceConfigs = new algo.Cache(makeServiceConfig),

  matchHost = (routeRules, host) => (
    routeRules && host && (
      (
        cfg = routeRules[host],
      ) => (
        !cfg && (
          (
            dot = host.indexOf('.'),
            wildcard,
          ) => (
            dot > 0 && (
              wildcard = '*' + host.substring(dot),
              cfg = routeRules[wildcard]
            ),
            !cfg && (
              cfg = routeRules['*']
            )
          )
        )(),
        cfg
      )
    )()
  ),

  hostHandlers = new algo.Cache(
    host => host && (
      (
        cfg = matchHost(config?.RouteRules?.[__port?.Port], host),
      ) => (
        cfg && (new algo.RoundRobinLoadBalancer(cfg))
      )
    )()
  ),

) => pipy({
  _balancer: null,
  _serviceConfig: null,
  _unhealthCache: null,
  _healthCheckTarget: null,
})

.export('tls-forward', {
  __service: null,
})

.import({
  __port: 'listener',
  __consumer: 'consumer',
  __cert: 'connect-tls',
  __target: 'connect-tcp',
  __metricLabel: 'connect-tcp',
  __healthCheckTargets: 'health-check',
  __healthCheckServices: 'health-check',
})

.pipeline()
.branch(
  () => __consumer?.sni, (
    $=>$
  )
)
.handleStreamStart(
  () => (
    (_balancer = hostHandlers.get(__consumer.sni)) && (
      (__service = serviceHandlers.get(_balancer.borrow({}).id)) && (
        (_serviceConfig = serviceConfigs.get(__service)) && (
          __metricLabel = __service.name,
          _unhealthCache = __healthCheckServices?.[__service.name],
          (__target = _serviceConfig.targetBalancer?.borrow?.({}, undefined, _unhealthCache)?.id) && (
            (
              attrs = _serviceConfig?.endpointAttributes?.[__target]
            ) => (
              attrs?.UpstreamCert ? (
                __cert = attrs?.UpstreamCert
              ) : (
                __cert = __service?.UpstreamCert
              )
            )
          )()
        )
      )
    )
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('[tls-forward] sni, target, cert:', __consumer.sni, __target, Boolean(__cert))
      )
    )
  )
)
.branch(
  () => !__target, (
    $=>$.replaceStreamStart(new StreamEnd)
  ),
  (
    $=>$.branch(
      () => __cert, (
        $=>$.use('lib/connect-tls.js')
      ), (
        $=>$.use('lib/connect-tcp.js')
      )
    )
    .handleStreamEnd(
      e => (
        (_healthCheckTarget = __healthCheckTargets?.[__target + '@' + __service.name]) && (
          (!e.error || e.error === "ReadTimeout" || e.error === "WriteTimeout" || e.error === "IdleTimeout") ? (
            _healthCheckTarget.service.ok(_healthCheckTarget)
          ) : (
            _healthCheckTarget.service.fail(_healthCheckTarget)
          )
        )
      )
    )
  )
)

)()