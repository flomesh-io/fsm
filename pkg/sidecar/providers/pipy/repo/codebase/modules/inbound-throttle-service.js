((
  { initRateLimit } = pipy.solve('utils.js'),
  { rateLimitCounter } = pipy.solve('metrics.js'),
  rateLimitedCounter = rateLimitCounter.withLabels('throttle_service'),
  rateLimitCache = new algo.Cache(initRateLimit),
) => (

pipy({
  _overflow: null,
  _rateLimit: null,
})

.import({
  __service: 'inbound-http-routing',
})

.pipeline()
.branch(
  () => _rateLimit = rateLimitCache.get(__service?.RateLimit), (
    $=>$.branch(
      () => _rateLimit.backlog > 0, (
        $=>$.branch(
          () => _rateLimit.count > _rateLimit.backlog, (
            $=>$
            .replaceData()
            .replaceMessage(
              () => (
                rateLimitedCounter.increase(),
                [_rateLimit.response, new StreamEnd]
              )
            )
          ), (
            $=>$
            .handleMessageStart(() => _rateLimit.count++)
            .throttleMessageRate(() => _rateLimit.quota, {blockInput: false})
            .chain()
            .handleMessageStart(() => _rateLimit.count--)
          )
        )
      ), (
        $=>$
        .branch(
          () => _rateLimit.quota.consume(1) !== 1, (
            $=>$.replaceMessage(
              () => (
                  rateLimitedCounter.increase(),
                  [_rateLimit.response, new StreamEnd]
              )
            )
          ), (
            $=>$.chain()
          )
        )
      )
    )
  ), (
    $=>$.chain()
  )
)
))()
