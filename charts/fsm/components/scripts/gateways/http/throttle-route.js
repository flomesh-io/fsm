((
  { initRateLimit } = pipy.solve('lib/utils.js'),
  { rateLimitCounter } = pipy.solve('lib/metrics.js'),
  rateLimitedCounter = rateLimitCounter.withLabels('throttle-route'),
  rateLimitCache = new algo.Cache(initRateLimit),
) => (

pipy({
  _overflow: null,
  _rateLimit: null,
  _localWaitCounter: null,
})

.import({
  __route: 'route',
})

.pipeline()
.branch(
  () => _rateLimit = rateLimitCache.get(__route?.config?.RateLimit), (
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
            .onStart(() => void(_localWaitCounter = { n: 0 }))
            .onEnd(() => void(_rateLimit.count -= _localWaitCounter.n))
            .handleMessageStart(() => (_rateLimit.count++, _localWaitCounter.n++))
            .throttleMessageRate(() => _rateLimit.quota, {blockInput: false})
            .chain()
            .handleMessageStart(() => (_rateLimit.count--, _localWaitCounter.n--))
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