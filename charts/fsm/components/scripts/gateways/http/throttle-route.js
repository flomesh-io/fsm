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