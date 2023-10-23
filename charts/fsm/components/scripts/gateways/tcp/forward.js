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
          failoverBalancer: serviceConfig.Endpoints && failover(Object.fromEntries(Object.entries(serviceConfig.Endpoints).map(([k, v]) => [k, v.Weight]))),
        },
      ) => (
        obj
      )
    )()
  ),

  serviceConfigs = new algo.Cache(makeServiceConfig),

  portHandlers = new algo.Cache(
    port => (
      (
        routeRules = config?.RouteRules?.[port],
      ) => (
        routeRules && (new algo.RoundRobinLoadBalancer(routeRules))
      )
    )()
  ),

) => pipy({
  _balancer: null,
  _serviceConfig: null,
  _unhealthCache: null,
  _healthCheckTarget: null,
})

.export('tcp-forward', {
  __service: null,
})

.import({
  __port: 'listener',
  __cert: 'connect-tls',
  __target: 'connect-tcp',
  __metricLabel: 'connect-tcp',
  __healthCheckTargets: 'health-check',
  __healthCheckServices: 'health-check',
})

.pipeline()
.handleStreamStart(
  () => (
    (_balancer = portHandlers.get(__port?.Port)) && (
      (__service = serviceHandlers.get(_balancer.borrow({}).id)) && (
        (_serviceConfig = serviceConfigs.get(__service)) && (
          __metricLabel = __service.name,
          _unhealthCache = __healthCheckServices?.[__service.name],
          (__target = _serviceConfig.targetBalancer?.borrow?.(__inbound, undefined, _unhealthCache)?.id) && (
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
        console.log('[tcp-forward] port, target, cert:', __port?.Port, __target, Boolean(__cert))
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
