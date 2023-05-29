((
  config = pipy.solve('config.js'),
  inboundL7Chains = config?.Chains?.["inbound-http"],
  inboundL4Chains = config?.Chains?.["inbound-tcp"],
  connectionLimitCounter = new stats.Counter('sidecar_local_rate_limit_inbound', ['label']),

  makePortHandler = port => (
    (
      portConfig = config?.Inbound?.TrafficMatches?.[port || 0],
      protocol = portConfig?.Protocol && (portConfig?.Protocol === 'http' || portConfig?.Protocol === 'grpc' ? 'http' : 'tcp'),
      isHTTP2 = portConfig?.Protocol === 'grpc',
      allowedEndpointsLocal = portConfig?.AllowedEndpoints,
      allowedEndpointsGlobal = config?.AllowedEndpoints || {},
      allow = (
        (
          allowedEndpoints = allowedEndpointsLocal
            ? Object.keys(allowedEndpointsLocal).filter(k => k in allowedEndpointsGlobal)
            : Object.keys(allowedEndpointsGlobal),
          ips = allowedEndpoints.filter(k => k.indexOf('/') < 0),
          ipSet = ips.length > 0 && new Set(ips),
          masks = allowedEndpoints.filter(k => k.indexOf('/') > 0),
          maskArray = masks.length > 0 && masks.map(e => new Netmask(e)),
        ) => (
          ipSet && maskArray ? (
            ip => (ipSet.has(ip) || maskArray.find(e => e.contains(ip)))
          ) : (
            ipSet ? (
              ip => ipSet.has(ip)
            ) : (
              maskArray ? (
                ip => maskArray.find(e => e.contains(ip))
              ) : () => false
            )
          )
        )
      )(),
      connectionQuota = portConfig?.RateLimit?.Local && (
        new algo.Quota(
          portConfig.RateLimit.Local?.Burst || portConfig.RateLimit.Local?.Connections || 0,
          {
            produce: portConfig.RateLimit.Local?.Connections || 0,
            per: portConfig.RateLimit.Local?.StatTimeWindow || 0,
          }
        )
      ),
      connectionLimit = connectionQuota && (
        (
          {
            namespace,
            pod,
          } = pipy.solve('utils.js'),
          label = namespace + '/' + pod.split('-')[0] + '_' + portConfig?.Port + '_' + portConfig?.Protocol,
        ) => (
          connectionLimitCounter.withLabels(label)
        )
      )(),
    ) => (
      !portConfig && (
        () => undefined
      ) || connectionQuota && (
        () => void (
          allow(__inbound.remoteAddress || '127.0.0.1') && (
            (connectionQuota.consume(1) === 1) || (connectionLimit.increase(), false)
          ) && (
            __port = portConfig,
            __protocol = protocol,
            __isHTTP2 = isHTTP2
          )
        )
      ) || (
        () => void (
          allow(__inbound.remoteAddress || '127.0.0.1') && (
            __port = portConfig,
            __protocol = protocol,
            __isHTTP2 = isHTTP2
          )
        )
      )
    )
  )(),

  portHandlers = new algo.Cache(makePortHandler),
) => pipy()

.export('inbound', {
  __port: null,
  __protocol: null,
  __isHTTP2: false,
  __isIngress: false,
})

.pipeline()
.onStart(
  () => void portHandlers.get(__inbound.destinationPort)()
)
.branch(
  () => __protocol === 'http', (
    $=>$
    .replaceStreamStart()
    .chain(inboundL7Chains)
    /*[
      'modules/inbound-tls-termination.js',
      'modules/inbound-http-routing.js',
      'modules/inbound-metrics-http.js',
      'modules/inbound-tracing-http.js',
      'modules/inbound-logging-http.js',
      'modules/inbound-throttle-service.js',
      'modules/inbound-throttle-route.js',
      'modules/inbound-http-load-balancing.js',
      'modules/inbound-http-default.js',
    ]*/
  ),

  () => __protocol == 'tcp', (
    $=>$.chain(inboundL4Chains)
    /*[
      'modules/inbound-tls-termination.js',
      'modules/inbound-tcp-routing.js',
      'modules/inbound-tcp-load-balancing.js',
      'modules/inbound-tcp-default.js',
    ]*/
  ),

  (
    $=>$.replaceStreamStart(
      new StreamEnd('ConnectionReset')
    )
  )
)

)()
