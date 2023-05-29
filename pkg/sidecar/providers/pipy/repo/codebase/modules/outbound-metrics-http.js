((
  {
    metricsCache,
    identityCache,
  } = pipy.solve('metrics.js'),
) => (

pipy({
  _requestTime: null
})

.import({
  __cluster: 'outbound-http-routing'
})

.pipeline()
.handleMessageStart(
  () => (
    _requestTime = Date.now()
  )
)
.chain()
.handleMessageStart(
  (msg) => (
    (
      clusterName = __cluster?.name,
      status = msg?.head?.status,
      statusClass = status / 100,
      metrics = metricsCache.get(clusterName),
      fsmRequestDurationHist = identityCache.get(msg?.head?.headers?.['fsm-stats']),
    ) => (
      fsmRequestDurationHist && (
        fsmRequestDurationHist.observe(Date.now() - _requestTime),
        delete msg.head.headers['fsm-stats']
      ),
      metrics.upstreamCompletedCount.increase(),
      metrics.upstreamResponseTotal.increase(),
      status && (
        metrics.upstreamCodeCount.withLabels(status).increase(),
        metrics.upstreamCodeXCount.withLabels(statusClass).increase(),
        metrics.upstreamResponseCode.withLabels(statusClass).increase()
      )
    )
  )()
)

))()