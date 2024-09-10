export default function (config) {
  var key = config.key
  var requests = Number.parseInt(config.rateLimit.requests)
  var interval = Number.parseFloat(config.rateLimit.interval)
  var burst = Number.parseInt(config.rateLimit.burst)
  var backlog = Number.parseInt(config.rateLimit.backlog)

  var response = new Message(
    {
      status: config.rateLimit.response?.status || 429,
      headers: config.rateLimit.response?.headers,
    },
    config.rateLimit.response?.body || 'Too many requests'
  )

  var rateQuota = new algo.Quota(burst || requests, {
      key: key ? `rate:${key}` : undefined,
      produce: requests,
      per: interval,
    }
  )

  var backlogQuota = new algo.Quota(backlog || 0, {
    key: key ? `backlog:${key}` : undefined
  })

  return pipeline($=>$
    .pipe(
      evt => {
        if (evt instanceof MessageStart) {
          if (backlog && backlogQuota.consume(1) === 0) {
            return 'deny'
          } else {
            return 'pass'
          }
        }
      }, {
        'pass': ($=>$
          .throttleMessageRate(rateQuota)
          .pipeNext()
          .handleMessageStart(() => {
            if (backlog) {
              backlogQuota.produce(1)
            }
          })
        ),
        'deny': ($=>$
          .replaceData()
          .replaceMessage(response)
        )
      }
    )
  )
}
