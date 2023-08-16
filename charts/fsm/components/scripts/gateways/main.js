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
  { config } = pipy.solve('config.js'),
  listeners = {},
  listenPort = 0,
) => pipy()

.export('listener', {
  __port: null,
})

.repeat(
  (config.Listeners || []),
  ($, l)=>$.listen(
    (listenPort = (l.Listen || l.Port || 0), listeners[listenPort] = new ListenerArray, listeners[listenPort].add(listenPort), listeners[listenPort]),
    { ...l, protocol: (l?.Protocol === 'UDP') ? 'udp' : 'tcp' }
  )
  .onStart(
    () => (
      __port = l,
      new Data
    )
  )
  .link('launch')
)

.pipeline('launch')
.branch(
  () => (__port?.Protocol === 'HTTP'), (
    $=>$.chain(config?.Chains?.HTTPRoute || [])
  ),
  () => (__port?.Protocol === 'HTTPS'), (
    $=>$.chain(config?.Chains?.HTTPSRoute || [])
  ),
  () => (__port?.Protocol === 'TLS' && __port?.TLS?.TLSModeType === 'Passthrough'), (
    $=>$.chain(config?.Chains?.TLSPassthrough || [])
  ),
  () => (__port?.Protocol === 'TLS' && __port?.TLS?.TLSModeType === 'Terminate'), (
    $=>$.chain(config?.Chains?.TLSTerminate || [])
  ),
  () => (__port?.Protocol === 'TCP'), (
    $=>$.chain(config?.Chains?.TCPRoute || [])
  ),
  (
    $=>$.replaceStreamStart(new StreamEnd)
  )
)

.task()
.onStart(new Data)
.use('common/health-check.js')

)()
