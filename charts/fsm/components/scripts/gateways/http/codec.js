((
  { metrics } = pipy.solve('lib/metrics.js'),
  acceptedMetric = metrics.fgwHttpCurrentConnections.withLabels('accepted'),
  activeMetric = metrics.fgwHttpCurrentConnections.withLabels('active'),
  handledMetric = metrics.fgwHttpCurrentConnections.withLabels('handled'),
  fgwHttpRequestsTotal = metrics.fgwHttpRequestsTotal,

) => pipy()

.export('http', {
  __http: null,
  __request: null,
  __response: null,
})

.pipeline()
.handleStreamStart(
  () => (
    acceptedMetric.increase(),
    activeMetric.increase()
  )
)
.handleStreamEnd(
  () => (
    handledMetric.increase(),
    activeMetric.decrease()
  )
)
.demuxHTTP().to(
  $=>$
  .handleMessageStart(
    msg => (
      __http = msg?.head,
      __request = { head: msg?.head, reqTime: Date.now() },
      fgwHttpRequestsTotal.increase()
    )
  )
  .handleMessageEnd(
    msg => __request.tail = msg.tail
  )
  .chain()
)

)()