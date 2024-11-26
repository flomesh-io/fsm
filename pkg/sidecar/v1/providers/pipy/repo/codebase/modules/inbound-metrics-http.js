((
  {
    identity,
    metricsCache,
  } = pipy.solve('metrics.js'),
) => (

pipy()

.import({
  __cluster: 'inbound-http-routing',
})

.pipeline()
.chain()
.handleMessageStart(
  (msg) => (
    (
      headers = msg?.head?.headers,
      metrics = metricsCache.get(__cluster?.name),
    ) => (
      headers && (
        headers['fsm-stats'] = identity,
        metrics.upstreamResponseTotal.increase(),
        metrics.upstreamResponseCode.withLabels(msg?.head?.status / 100).increase()
      )
    )
  )()
)

))()