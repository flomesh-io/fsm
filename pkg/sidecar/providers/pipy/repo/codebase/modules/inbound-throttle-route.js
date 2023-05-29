((
  { initRateLimit } = pipy.solve('utils.js'),
  { rateLimitCounter } = pipy.solve('metrics.js'),
  rateLimitedCounter = rateLimitCounter.withLabels('throttle-route'),
  rateLimitCache = new algo.Cache(initRateLimit),
) => (

pipy({
  _overflow: null,
  _rateLimit: null,
})

.import({
  __route: 'inbound-http-routing',
})

.pipeline()
.branch(
  () => _rateLimit = rateLimitCache.get(__route?.RateLimit), (
    $=>$.branch(
      () => _rateLimit.backlog > 0, (
        $=>$.mux(() => _rateLimit, () => ({ maxQueue: _rateLimit.backlog })).to(
          $=>$
          .onStart(({ sessionCount }) => void (_overflow = (sessionCount > 1)))
          .branch(
            () => _overflow, (
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
              .throttleMessageRate(() => _rateLimit.quota)
              .demux().to($=>$.chain())
            )
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
