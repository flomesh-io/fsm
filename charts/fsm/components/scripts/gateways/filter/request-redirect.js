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

  makeRedirectHandler = cfg => (
    msg => cfg?.statusCode ? (
      (
        scheme = cfg?.scheme || msg?.scheme || 'http',
        hostname = cfg?.hostname || msg?.host,
        path = cfg?.path || msg?.path,
        port = cfg?.port,
      ) => (
        port && hostname && (
          hostname = hostname.split(':')[0] + ':' + port
        ),
        hostname && path ? (
          new Message({
            status: cfg.statusCode,
            headers: {
              Location: scheme + '://' + hostname + path
            }
          })
        ) : null
      )
    )() : null
  ),

  redirectHandlers = new algo.Cache(makeRedirectHandler),

  makeServiceRedirectHandler = svc => (
    (svc?.Filters || []).filter(
      e => e?.Type === 'RequestRedirect'
    ).map(
      e => redirectHandlers.get(e)
    ).filter(
      e => e
    )?.[0]
  ),

  serviceRedirectHandlers = new algo.Cache(makeServiceRedirectHandler),

) => pipy({
  _redirectHandler: null,
  _redirectMessage: null,
 })

.import({
  __service: 'service',
})

.pipeline()
.onStart(
  () => void (
    _redirectHandler = serviceRedirectHandlers.get(__service)
  )
)
.branch(
  () => _redirectHandler, (
    $=>$.handleMessageStart(
      msg => (
        _redirectMessage = _redirectHandler(msg?.head?.headers)
      )
    )
    .branch(
      () => _redirectMessage, (
        $=>$.replaceMessage(
          msg => (
            isDebugEnabled && (
              console.log('[request-redirect] messages:', msg, _redirectMessage)
            ),
            _redirectMessage
          )
        )
      ), (
        $=>$.chain()
      )
    )
  ), (
    $=>$.chain()
  )
)

)()