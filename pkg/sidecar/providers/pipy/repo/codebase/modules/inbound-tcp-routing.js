((
  config = pipy.solve('config.js'),

  clusterCache = new algo.Cache(
    (clusterName => (
      (cluster = config?.Inbound?.ClustersConfigs?.[clusterName]) => (
        cluster ? Object.assign({ name: clusterName, Endpoints: cluster }) : null
      )
    )())
  ),

  clusterBalancers = new algo.Cache(cluster => new algo.RoundRobinLoadBalancer(cluster || {})),
) => pipy({
  _clusterName: null,
})

.import({
  __port: 'inbound',
})

.export('inbound-tcp-routing', {
  __cluster: null,
})

.pipeline()
.handleStreamStart(
  () => (
    (_clusterName = clusterBalancers.get(__port?.TcpServiceRouteRules?.TargetClusters)?.borrow?.()?.id) && (
      __cluster = clusterCache.get(_clusterName)
    )
  )
)
.chain()

)()