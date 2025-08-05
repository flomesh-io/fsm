var latencyCount = new stats.Counter('fgw_dubbo_latency_count')
var latencyTotal = new stats.Counter('fgw_dubbo_latency_total', ['scope'])
var latencyTotalUpstream = latencyTotal.withLabels('upstream')
var latencyTotalProxy = latencyTotal.withLabels('proxy')
var latencyTotalRequest = latencyTotal.withLabels('request')

export default function (config) {
  var conf = config.callDubbo
  var version = conf.version || ''
  var service = conf.service || ''
  var method = conf.method || ''
  var signature = conf.signature || ''

  var $ctx
  var $headTime

  var requestID = 0

  var pl = pipeline($=>$
    .onStart(c => { $ctx = c })
    .replaceMessage(
      req => {
        var body = req.body
        var json = body.size > 0 ? JSON.decode(req.body) : []
        var params = (json instanceof Array ? json : [json])
        return new Message(
          {
            requestID: ++requestID,
            isRequest: true,
            isTwoWay: true,
            serializationType: 2,
          },
          Hessian.encode([
            '2.0.2', service, version, method, signature, ...params, null
          ])
        )
      }
    )
    .pipeNext()
    .handleMessageStart(() => { $headTime = pipy.performance.now() })
    .replaceMessage(
      res => {
        var results = Hessian.decode(res.body)
        var msg = new Message(JSON.encode(results))
        var response = $ctx.response
        var timeRequest = $headTime - $ctx.headTime
        var timeUpstream = response.headTime - $ctx.sendTime
        var timeProxy = timeRequest - timeUpstream
        latencyCount.increase()
        latencyTotalUpstream.increase(timeUpstream)
        latencyTotalProxy.increase(timeProxy)
        latencyTotalRequest.increase(timeRequest)
        return msg
      }
    )
  )

  pl.meta = { outputProtocol: 'dubbo' }
  return pl
}
