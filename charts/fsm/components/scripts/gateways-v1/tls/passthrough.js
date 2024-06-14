((
  { config, isDebugEnabled } = pipy.solve('config.js'),

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
    host => (
      (
        upstream = matchHost(config?.RouteRules?.[__port?.Port], host),
        target,
        defaultPort = config?.Configs?.DefaultPassthroughUpstreamPort || '443'
      ) => (
        upstream ? (
          upstream.startsWith('[') ? (
            upstream.indexOf(']:') > 0 ? (
              target = upstream
            ) : (
              target = upstream + ':' + defaultPort
            )
          ) : (
            upstream.indexOf(':') > 0 ? (
              target = upstream
            ) : (
              target = upstream + ':' + defaultPort
            )
          )
        ) : (
          target = null
        ),
        target
      )
    )()
  ),

) => pipy({
  _sni: undefined,
  _passthroughTarget: undefined,
})

.import({
  __port: 'listener',
  __consumer: 'consumer',
  __target: 'connect-tcp',
  __metricLabel: 'connect-tcp',
})

.pipeline()
.handleTLSClientHello(
  hello => (
    _sni = hello?.serverNames?.[0] || '',
    _passthroughTarget = hostHandlers.get(_sni),
    __consumer = {sni: _sni, target: _passthroughTarget, type: 'passthrough'},
    __target = __metricLabel = _passthroughTarget
  )
)
.branch(
  () => _passthroughTarget || (_passthroughTarget === null), (
    $=>$
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('[https-passthrough] port, sni, target:', __port?.Port, _sni, __target)
      )
    )
  )
)
.chain()
.branch(
  () => Boolean(_passthroughTarget), (
    $=>$.use('lib/connect-tcp.js')
  ), (
    $=>$.replaceStreamStart(new StreamEnd)
  )
)

)()
