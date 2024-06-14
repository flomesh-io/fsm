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
})

.export('udp-forward', {
  __service: null,
})

.import({
  __port: 'listener',
  __target: 'connect-udp',
  __metricLabel: 'connect-udp',
})

.pipeline()
.handleStreamStart(
  () => (
    (_balancer = portHandlers.get(__port?.Port)) && (
      (__service = serviceHandlers.get(_balancer.borrow({})?.id)) && (
        (_serviceConfig = serviceConfigs.get(__service)) && (
          __metricLabel = __service.name,
          (__target = _serviceConfig.targetBalancer?.borrow?.(__inbound, undefined, undefined)?.id)
        )
      )
    )
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('[udp-forward] port, target:', __port?.Port, __target)
      )
    )
  )
)
.branch(
  () => __target, (
    $=>$.use('lib/connect-udp.js')
  ), (
    $=>$.replaceStreamStart(
      () => (
        new StreamEnd('ConnectionReset')
      )
    )
  )
)

)()