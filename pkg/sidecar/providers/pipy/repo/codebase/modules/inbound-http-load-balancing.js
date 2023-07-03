((
  config = pipy.solve('config.js'),
  isDebugEnabled = config?.Spec?.SidecarLogLevel === 'debug',
  targetBalancers = new algo.Cache(cluster => new algo.RoundRobinLoadBalancer(cluster?.Endpoints || {})),
) => pipy({
  _targetObject: null,
})

.import({
  __port: 'inbound',
  __isHTTP2: 'inbound',
  __isIngress: 'inbound',
  __route: 'inbound-http-routing',
  __service: 'inbound-http-routing',
  __cluster: 'inbound-http-routing',
  __target: 'connect-tcp',
  __metricLabel: 'connect-tcp',
})

.pipeline()
.handleStreamStart(
  () => (
    (_targetObject = targetBalancers.get(__cluster)?.borrow?.()) && (
      __metricLabel = __cluster.name,
      __target = _targetObject.id
    )
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('inbound-http # port/service/route/cluster/ingress :',
          __port?.Port, __service?.name, __route?.Path, __cluster?.name, __isIngress)
      )
    )
  )
)
.branch(
  () => !__target, (
    $=>$.chain()
  ),

  (
    $=>$.muxHTTP(() => _targetObject, { version: () => __isHTTP2 ? 2 : 1 }).to(
      $=>$.use('connect-tcp.js')
    )
  )
)

)()