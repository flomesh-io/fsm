((
  config = pipy.solve('config.js'),

  certChain = config?.Certificate?.CertChain,

  forwardMatches = config?.Forward?.ForwardMatches && Object.fromEntries(
    Object.entries(config.Forward.ForwardMatches).map(
      ([k, v]) => [
        k, new algo.RoundRobinLoadBalancer(v || {})
      ]
    )
  ),

  forwardEgressGateways = config?.Forward?.EgressGateways && Object.fromEntries(
    Object.entries(config.Forward.EgressGateways).map(
      ([k, v]) => [
        k, { balancer: new algo.RoundRobinLoadBalancer(
          Object.fromEntries(Object.entries(v?.Endpoints || {}).map(
            ([k, v]) => [k, v?.Weight || 100]
          ))
        ), mode: v?.Mode }
      ]
    )
  ),
) => (

pipy({
  _origTarget: null,
  _egressType: '',
  _egressEndpoint: null,
})

.import({
  __port: 'outbound',
  __cert: 'outbound',
  __isEgress: 'outbound',
  __target: 'connect-tcp',
})

.pipeline()
.onStart(
  () => void (
    forwardMatches && (
      (
        egw = forwardMatches[__port?.EgressForwardGateway || '*']?.next?.()?.id,
      ) => (
        egw && (
          _egressType = forwardEgressGateways?.[egw]?.mode || 'http2tunnel',
          _egressEndpoint = forwardEgressGateways?.[egw]?.balancer?.next?.()?.id
        )
      )
    )()
  )
)
.branch(
  () => __cert || (certChain && !__isEgress), (
    $=>$.use('connect-tls.js')
  ),
  () => __isEgress && _egressEndpoint, (
    $=>$
    .handleStreamStart(
      () => (
        _origTarget = __target,
        __target = _egressEndpoint
      )
    )
    .branch(
      () => _egressType === 'http2tunnel', (
        $=>$.connectHTTPTunnel(
          () => new Message({
            method: 'CONNECT',
            path: _origTarget,
          })
        ).to($=>$.muxHTTP(() => _origTarget, { version: 2 }).to($=>$.use('connect-tcp.js')))
      ),
      (
        $=>$.connectSOCKS(
          () => _origTarget,
        ).to($=>$.use('connect-tcp.js'))
      )
    )
  ),
  (
    $=>$.use('connect-tcp.js')
  )
)

))()