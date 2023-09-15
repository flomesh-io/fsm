((
  { metrics } = pipy.solve('lib/metrics.js'),
) => pipy()

.export('http', {
  __http: null,
  __request: null,
  __response: null,
})

.pipeline()
.handleStreamStart(
  () => (
    metrics.fgwHttpCurrentConnections.withLabels('accepted').increase(),
    metrics.fgwHttpCurrentConnections.withLabels('active').increase()
  )
)
.handleStreamEnd(
  () => (
    metrics.fgwHttpCurrentConnections.withLabels('handled').increase(),
    metrics.fgwHttpCurrentConnections.withLabels('active').decrease()
  )
)
.demuxHTTP().to(
  $=>$
  .handleMessageStart(
    msg => (
      __http = msg?.head,
      __request = { head: msg?.head, reqTime: Date.now() },
      metrics.fgwHttpRequestsTotal.increase()
    )
  )
  .handleMessageEnd(
    msg => __request.tail = msg.tail
  )
  .chain()
)

)()