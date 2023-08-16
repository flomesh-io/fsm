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
    {
      config,
      tlsDomains,
      tlsWildcardDomains
    } = pipy.solve('config.js'),

  ) => pipy({
    _reject: false
  })

  .import({
    __route: 'main',
    __isTLS: 'main'
  })

  .pipeline()
    .handleMessageStart(
      msg => (
        ((host, hostname)  => (
          host = msg.head.headers['host'],
          hostname = host ? host.split(":")[0] : '',

          console.log("[reject-http] hostname", hostname),
          console.log("[reject-http] __isTLS", __isTLS),

          !__isTLS && (
            _reject = (
              Boolean(tlsDomains.find(domain => domain === hostname)) ||
              Boolean(tlsWildcardDomains.find(domain => domain.test(hostname)))
            )
          ),
          console.log("[reject-http] _reject", _reject)
        ))()
      )
    )
    .branch(
      () => (_reject), (
        $ => $
          .replaceMessage(
            new Message({
              "status": 403,
              "headers": {
                "Server": "pipy/0.70.0"
              }
            }, 'Forbidden')
          )
      ), (
        $=>$.chain()
      )
    )
)()
