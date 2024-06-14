((
  { metrics } = pipy.solve('lib/metrics.js'),
  acceptedMetric = metrics.fgwHttpCurrentConnections.withLabels('accepted'),
  activeMetric = metrics.fgwHttpCurrentConnections.withLabels('active'),
  handledMetric = metrics.fgwHttpCurrentConnections.withLabels('handled'),
  fgwHttpRequestsTotal = metrics.fgwHttpRequestsTotal,

) => pipy()

.export('http', {
  __http: null,
  __requestHead: null,
  __requestTail: null,
  __requestTime: null,
  __responseHead: null,
  __responseTail: null,
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
      __http = msg.head,
      __requestHead = msg.head,
      __requestTime = Date.now(),
      fgwHttpRequestsTotal.increase()
    )
  )
  .handleMessageEnd(
    msg => __requestTail = msg.tail
  )
  .chain()
)

)()