export default function (config) {
  var delayPercentage = config.faultInjection.delay?.percentage | 0
  var delayMin = config.faultInjection.delay?.min | 0
  var delayMax = config.faultInjection.delay?.max | 0
  var abortPercentage = config.faultInjection.abort?.percentage | 0

  delayPercentage = Math.max(0, Math.min(100, delayPercentage))
  abortPercentage = Math.max(0, Math.min(100, abortPercentage))

  var abortMessage = new Message(
    {
      status: config.faultInjection.abort?.response?.status || 503,
      headers: config.faultInjection.abort?.response?.headers,
    },
    config.faultInjection.abort?.response?.body || 'Service unavailable'
  )

  var delayDispatcher = new algo.LoadBalancer([true, false], {
    weight: target => target ? delayPercentage : 100 - delayPercentage
  })

  var abortDispatcher = new algo.LoadBalancer([true, false], {
    weight: target => target ? abortPercentage : 100 - abortPercentage
  })

  return pipeline($=>$
    .handleMessageStart(
      () => {
        if (delayDispatcher.allocate().target) {
          return new Timeout((Math.random() * (delayMax - delayMin) + delayMin) / 1000).wait()
        }
      }
    )
    .pipe(() => abortDispatcher.allocate().target ? 'abort' : 'pass', {
      'abort': $=>$.replaceData().replaceMessage(abortMessage),
      'pass': $=>$.pipeNext()
    })
  )
}
