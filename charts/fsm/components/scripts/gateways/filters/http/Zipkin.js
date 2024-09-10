export default function (config) {
  var samplePercentage = Number.parseInt(config.zipkin.samplePercentage) || 100

  var sampleDecider = new algo.LoadBalancer([true, false], {
    weight: target => target ? samplePercentage : 100 - samplePercentage
  })

  return pipeline($=>$
    .handleMessageStart(
      msg => {
        var headers = msg.head.headers
        var traceId = headers['x-b3-traceid']
        var spanId = headers['x-b3-spanid']
        var sampled = headers['x-b3-sampled']
        var randomId = algo.uuid().replaceAll('-', '')

        if (spanId) headers['x-b3-parentspanid'] = spanId
        if (!traceId) headers['x-b3-traceid'] = randomId
        if (!sampled) headers['x-b3-sampled'] = sampleDecider.allocate().target ? 1 : 0

        headers['x-b3-spanid'] = randomId.substring(16)
        headers['x-b3-sampled']
      }
    )
    .pipeNext()
  )
}
