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
    tracingEnabled,
    initTracingHeaders,
    makeZipKinData,
    saveTracing,
  } = pipy.solve('lib/tracing.js'),
  sampledCounter0 = new stats.Counter('fgw_tracing_sampled_0'),
  sampledCounter1 = new stats.Counter('fgw_tracing_sampled_1'),
) => (

pipy({
  _sampled: true,
  _zipkinData: null,
  _httpBytesStruct: null,
})

.import({
  __port: 'listener',
  __service: 'service',
  __target: 'connect-tcp',
})

.pipeline()
.handleMessage(
  (msg) => (
    tracingEnabled && (
      (_sampled = initTracingHeaders(msg.head.headers, __port?.Protocol)) && (
        _httpBytesStruct = {},
        _httpBytesStruct.requestSize = msg?.body?.size,
        _zipkinData = makeZipKinData(msg, msg.head.headers, __service?.name)
      ),
      _sampled ? sampledCounter1.increase() : sampledCounter0.increase()
    )
  )
)
.chain()
.handleMessage(
  (msg) => (
    tracingEnabled && _sampled && (
      _httpBytesStruct.responseSize = msg?.body?.size,
      saveTracing(_zipkinData, msg?.head, _httpBytesStruct, __target)
    )
  )
)

))()