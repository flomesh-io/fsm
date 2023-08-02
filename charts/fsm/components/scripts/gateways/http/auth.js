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
  { config, isDebugEnabled } = pipy.solve('config.js'),

  headersAuthorization = {},

  _0 = (config?.Consumers || []).forEach(
    c => (
      Object.entries(c['Headers-Authorization'] || {}).map(
        ([k, v]) => (
          !headersAuthorization[k] && (headersAuthorization[k] = {}),
          headersAuthorization[k][v] = c
        )
      )
    )
  ),

) => pipy({
  _consumer: null,
})

.import({
  __consumer: 'consumer',
})

.pipeline()
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      msg => (
        Object.keys(headersAuthorization).forEach(
          h => !_consumer && (
            (_consumer = headersAuthorization[h][msg?.head?.headers?.[h]]) && (
              __consumer ? (
                Object.keys(_consumer).forEach(
                  k => (__consumer[k] = _consumer[k])
                )
              ) : (
                __consumer = _consumer
              )
            )
          )
        ),
        console.log('[auth] consumer, msg:', __consumer, msg)
      )
    )
  )
)
.chain()

)()