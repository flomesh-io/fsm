((
  circuitBreakers = {},

  makeCircuitBreaker = (clusterConfig) => (
      clusterConfig?.ConnectionSettings?.http?.CircuitBreaking && (circuitBreakers[clusterConfig.name] = (
        (
          clusterName = clusterConfig.name || '',
          minRequestAmount = clusterConfig.ConnectionSettings.http.CircuitBreaking?.MinRequestAmount || 100,
          statTimeWindow = clusterConfig.ConnectionSettings.http.CircuitBreaking?.StatTimeWindow || 30, // 30s
          slowTimeThreshold = clusterConfig.ConnectionSettings.http.CircuitBreaking?.SlowTimeThreshold || 5, // 5s
          slowAmountThreshold = clusterConfig.ConnectionSettings.http.CircuitBreaking?.SlowAmountThreshold || 0,
          slowRatioThreshold = clusterConfig.ConnectionSettings.http.CircuitBreaking?.SlowRatioThreshold || 0.0,
          errorAmountThreshold = clusterConfig.ConnectionSettings.http.CircuitBreaking?.ErrorAmountThreshold || 0,
          errorRatioThreshold = clusterConfig.ConnectionSettings.http.CircuitBreaking?.ErrorRatioThreshold || 0.0,
          degradedTimeWindow = clusterConfig.ConnectionSettings.http.CircuitBreaking?.DegradedTimeWindow || 30, // 30s
          degradedStatusCode = clusterConfig.ConnectionSettings.http.CircuitBreaking?.DegradedStatusCode || 409,
          degradedResponseContent = clusterConfig.ConnectionSettings.http.CircuitBreaking?.DegradedResponseContent || 'Coming soon ...',
          tick = 0,
          delay = 0,
          total = 0,
          slowAmount = 0,
          errorAmount = 0,
          degraded = false,
          lastDegraded = false,
          slowQuota = slowAmountThreshold > 0 ? (
            new algo.Quota(slowAmountThreshold - 1, {
              per: statTimeWindow
            })
          ) : null,
          errorQuota = errorAmountThreshold > 0 ? (
            new algo.Quota(errorAmountThreshold - 1, {
              per: statTimeWindow
            })
          ) : null,
          slowEnabled = slowQuota || (slowRatioThreshold > 0),
          errorEnabled = errorQuota || (errorRatioThreshold > 0),
          open,
          close,
        ) => (
          open = () => (
            console.log('[circuit_breaker] total/slowAmount/errorAmount (open) ', clusterName, total, slowAmount, errorAmount)
          ),

          close = () => (
            console.log('[circuit_breaker] total/slowAmount/errorAmount (close)', clusterName, total, slowAmount, errorAmount)
          ),

          {
            increase: () => (
              ++total
            ),

            isDegraded: () => (
              degraded
            ),

            checkSlow: seconds => (
              slowEnabled && (seconds >= slowTimeThreshold) && (
                lastDegraded = degraded,
                (slowAmount < total) && ++slowAmount,
                slowQuota && (slowQuota.consume(1) != 1) && (degraded = true) || (
                  (slowRatioThreshold > 0) && (total >= minRequestAmount) && (slowAmount / total >= slowRatioThreshold) && (
                    degraded = true
                  )
                ),
                !lastDegraded && degraded && open()
              )
            ),

            checkError: statusCode => (
              errorEnabled && (statusCode > 499) && (
                lastDegraded = degraded,
                (errorAmount < total) && ++errorAmount,
                errorQuota && (errorQuota.consume(1) != 1) && (degraded = true) || (
                  (errorRatioThreshold > 0) && (total >= minRequestAmount) && (errorAmount / total >= errorRatioThreshold) && (
                    degraded = true
                  )
                ),
                !lastDegraded && degraded && open()
              )
            ),

            refreshTimer: () => (
              degraded && (
                (++delay > degradedTimeWindow) && (
                  lastDegraded = degraded = false,
                  close(),
                  delay = tick = total = slowAmount = errorAmount = 0
                )
              ),
              !degraded && (
                (++tick > statTimeWindow) && (
                  tick = total = slowAmount = errorAmount = 0
                )
              )
            ),

            message: () => (
              [
                new Message({ status: degradedStatusCode }, degradedResponseContent),
                new StreamEnd
              ]
            ),
          }
        )
      )())
  ),

  circuitBreakerCache = new algo.Cache(makeCircuitBreaker),

) => pipy({
  _requestTime: null,
  _circuitBreaker: null,
})

.import({
  __cluster: 'outbound-http-routing',
})

.pipeline()
.branch(
  () => _circuitBreaker = circuitBreakerCache.get(__cluster), (
    $=>$
    .branch(
      () => _circuitBreaker.isDegraded(), (
        $=>$.replaceMessage(
          () => _circuitBreaker.message()
        )
      ), (
        $=>$
        .handleMessageStart(
          () => (
            _requestTime = Date.now(),
            _circuitBreaker.increase()
          )
        )
        .chain()
        .handleMessageStart(
          msg => (
            _circuitBreaker.checkError(msg.head.status),
            _circuitBreaker.checkSlow((Date.now() - _requestTime) / 1000)
          )
        )
      )
    )
  ), (
    $=>$.chain()
  )
)

.task('1s')
.onStart(
  () => new Message
)
.replaceMessage(
  () => (
    Object.values(circuitBreakers).forEach(
      obj => obj.refreshTimer()
    ),
    new StreamEnd
  )
)

)()
