((
  config = pipy.solve('config.js'),
  isDebugEnabled = config?.Spec?.SidecarLogLevel === 'debug',
  targetBalancers = new algo.Cache(target => new algo.RoundRobinLoadBalancer(target?.Endpoints || {})),
) => pipy()

.import({
  __port: 'inbound',
  __cluster: 'inbound-tcp-routing',
  __target: 'connect-tcp',
  __metricLabel: 'connect-tcp',
})

.pipeline()
.handleStreamStart(
  () => (
    (__target = __cluster && targetBalancers.get(__cluster)?.next?.()?.id) && (
      __metricLabel = __cluster.name
    )
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('inbound-tcp # port/cluster :', __port?.Port, __cluster?.name)
      )
    )
  )
)
.branch(
  () => !__target, (
    $=>$.chain()
  ),

  (
    $=>$.use('connect-tcp.js')
  )
)

)()