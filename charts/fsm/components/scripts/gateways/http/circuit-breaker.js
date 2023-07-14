/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

((
  { isDebugEnabled } = pipy.solve('config.js'),

  circuitBreakers = {},

  makeCircuitBreaker = (serviceConfig) => (
      serviceConfig?.ConnectionSettings?.http?.CircuitBreaking && (circuitBreakers[serviceConfig.name] = (
        (
          serviceName = serviceConfig.name || '',
          minRequestAmount = serviceConfig.ConnectionSettings.http.CircuitBreaking?.MinRequestAmount || 100,
          statTimeWindow = serviceConfig.ConnectionSettings.http.CircuitBreaking?.StatTimeWindow || 30, // 30s
          slowTimeThreshold = serviceConfig.ConnectionSettings.http.CircuitBreaking?.SlowTimeThreshold || 5, // 5s
          slowAmountThreshold = serviceConfig.ConnectionSettings.http.CircuitBreaking?.SlowAmountThreshold || 0,
          slowRatioThreshold = serviceConfig.ConnectionSettings.http.CircuitBreaking?.SlowRatioThreshold || 0.0,
          errorAmountThreshold = serviceConfig.ConnectionSettings.http.CircuitBreaking?.ErrorAmountThreshold || 0,
          errorRatioThreshold = serviceConfig.ConnectionSettings.http.CircuitBreaking?.ErrorRatioThreshold || 0.0,
          degradedTimeWindow = serviceConfig.ConnectionSettings.http.CircuitBreaking?.DegradedTimeWindow || 30, // 30s
          degradedStatusCode = serviceConfig.ConnectionSettings.http.CircuitBreaking?.DegradedStatusCode || 409,
          degradedResponseContent = serviceConfig.ConnectionSettings.http.CircuitBreaking?.DegradedResponseContent || 'Coming soon ...',
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
            console.log('[circuit_breaker] total/slowAmount/errorAmount (open) ', serviceName, total, slowAmount, errorAmount)
          ),

          close = () => (
            console.log('[circuit_breaker] total/slowAmount/errorAmount (close)', serviceName, total, slowAmount, errorAmount)
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
  __service: 'service',
})

.pipeline()
.onStart(
  () => void (
    _circuitBreaker = circuitBreakerCache.get(__service)
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('[circuit-breaker] service, _circuitBreaker:', __service.name, Boolean(_circuitBreaker))
      )
    )
  )
)
.branch(
  () => _circuitBreaker, (
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