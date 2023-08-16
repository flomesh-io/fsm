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

  matchHost = (routeRules, host) => (
    routeRules && host && (
      (
        cfg = routeRules[host],
      ) => (
        !cfg && (
          (
            dot = host.indexOf('.'),
            wildcard,
          ) => (
            dot > 0 && (
              wildcard = '*' + host.substring(dot),
              cfg = routeRules[wildcard]
            ),
            !cfg && (
              cfg = routeRules['*']
            )
          )
        )(),
        cfg
      )
    )()
  ),

  hostHandlers = new algo.Cache(
    host => (
      (
        upstream = matchHost(config?.RouteRules?.[__port?.Port], host),
        target,
        defaultPort = config?.Configs?.DefaultPassthroughUpstreamPort || '443'
      ) => (
        upstream ? (
          upstream.startsWith('[') ? (
            upstream.indexOf(']:') > 0 ? (
              target = upstream
            ) : (
              target = upstream + ':' + defaultPort
            )
          ) : (
            upstream.indexOf(':') > 0 ? (
              target = upstream
            ) : (
              target = upstream + ':' + defaultPort
            )
          )
        ) : (
          target = null
        ),
        target
      )
    )()
  ),

) => pipy({
  _sni: undefined,
  _passthroughTarget: undefined,
})

.import({
  __port: 'listener',
  __consumer: 'consumer',
  __target: 'connect-tcp',
  __metricLabel: 'connect-tcp',
})

.pipeline()
.handleTLSClientHello(
  hello => (
    _sni = hello?.serverNames?.[0] || '',
    _passthroughTarget = hostHandlers.get(_sni),
    __consumer = {sni: _sni, target: _passthroughTarget, type: 'passthrough'},
    __target = __metricLabel = _passthroughTarget
  )
)
.branch(
  () => _passthroughTarget || (_passthroughTarget === null), (
    $=>$
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('[https-passthrough] port, sni, target:', __port?.Port, _sni, __target)
      )
    )
  )
)
.chain()
.branch(
  () => Boolean(_passthroughTarget), (
    $=>$.use('lib/connect-tcp.js')
  ), (
    $=>$.replaceStreamStart(new StreamEnd)
  )
)

)()
