((
  config = pipy.solve('config.js'),
  specEnableEgress = config?.Spec?.Traffic?.EnableEgress,
  isDebugEnabled = config?.Spec?.SidecarLogLevel === 'debug',

  targetBalancers = new algo.Cache(target => new algo.RoundRobinLoadBalancer(
    Object.fromEntries(Object.entries(target?.Endpoints || {}).map(([k, v]) => [k, v.Weight || 100]))
  )),
) => pipy()

.import({
  __port: 'outbound',
  __cert: 'outbound',
  __isEgress: 'outbound',
  __cluster: 'outbound-tcp-routing',
  __metricLabel: 'connect-tcp',
  __target: 'connect-tcp',
})

.pipeline()
.handleStreamStart(
  () => (
    __target = __cluster && targetBalancers.get(__cluster)?.borrow?.()?.id,
    !__target && (specEnableEgress || __port?.TcpServiceRouteRules?.AllowedEgressTraffic) && (
      __target = __inbound.destinationAddress + ':' + __inbound.destinationPort,
      __cluster = {name: __target},
      __isEgress = true
    ),
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
        console.log('outbound-tcp # port/cluster/egress :', __port?.Port, __cluster?.name, __isEgress)
      )
    )
  )
)
.branch(
  () => !__target, (
    $=>$.chain()
  ),
  (
    $=>$.use('connect-upstream.js')
  )
)

)()