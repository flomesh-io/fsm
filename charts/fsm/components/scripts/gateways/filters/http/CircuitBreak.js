export default function (config) {
  var $requestTime

  var key = config.key
  var latencyThreshold = Number.parseFloat(config.circuitBreak.latencyThreshold)
  var errorCountThreshold = Number.parseInt(config.circuitBreak.errorCountThreshold)
  var errorRatioThreshold = Number.parseFloat(config.circuitBreak.errorRatioThreshold)
  var concurrencyThreshold = Number.parseInt(config.circuitBreak.concurrencyThreshold)
  var checkInterval = Number.parseFloat(config.circuitBreak.checkInterval) * 1000
  var breakInterval = Number.parseFloat(config.circuitBreak.breakInterval) * 1000

  var response = new Message(
    {
      status: config.circuitBreak.response?.status || 503,
      headers: config.circuitBreak.response?.headers,
    },
    config.circuitBreak.response?.body || 'Service unavailable'
  )

  var sharedStates = new algo.SharedMap(key)

  sharedStates.set('start', 0)
  sharedStates.set('concurrency', 0)
  sharedStates.set('total', 0)
  sharedStates.set('error', 0)
  sharedStates.set('brokenBefore', 0)

  function isBroken() {
    var brokenBefore = sharedStates.get('brokenBefore')
    return (brokenBefore && Date.now() < brokenBefore)
  }

  function check() {
    var t = Date.now()
    var total = sharedStates.add('total', 1)
    if (t - $requestTime >= latencyThreshold) {
      var error = sharedStates.add('error', 1)
    } else {
      var error = sharedStates.get('error')
    }
    var concurrency = sharedStates.get('concurrency')
    if (
      total > 0 &&
      concurrency >= concurrencyThreshold &&
      error >= errorCountThreshold &&
      error / total >= errorRatioThreshold
    ) {
      sharedStates.set('brokenBefore', t + breakInterval)
      sharedStates.set('start', 0)
      sharedStates.set('total', 0)
      sharedStates.set('error', 0)
    } else {
      var start = sharedStates.get('start')
      if (start && t - start >= checkInterval) {
        sharedStates.set('start', t)
        sharedStates.set('total', 0)
        sharedStates.set('error', 0)
      } else {
        sharedStates.set('start', t)
      }
    }
  }

  return pipeline($=>$
    .pipe(
      evt => {
        if (evt instanceof MessageStart) {
          if (isBroken()) {
            return 'deny'
          } else {
            $requestTime = Date.now()
            sharedStates.add('concurrency', 1)
            return 'pass'
          }
        }
      }, {
        'pass': ($=>$
          .pipeNext()
          .handleMessageStart(
            () => {
              sharedStates.sub('concurrency', 1)
              check()
            }
          )
        ),
        'deny': ($=>$
          .replaceData()
          .replaceMessage(response)
        )
      }
    )
  )
}
