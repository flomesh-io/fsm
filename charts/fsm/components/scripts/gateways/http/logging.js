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
    loggingEnabled,
    makeLoggingData,
    saveLoggingData,
  } = pipy.solve('lib/logging.js'),
  sampledCounter0 = new stats.Counter('inbound_http_logging_sampled_0'),
  sampledCounter1 = new stats.Counter('inbound_http_logging_sampled_1'),
) => (

pipy({
  _loggingData: null
})

.import({
  __service: 'service',
  __target: 'connect-tcp',
})

.pipeline()
.handleMessage(
  (msg) => (
    loggingEnabled && (
      _loggingData = makeLoggingData(msg, __inbound.remoteAddress, __inbound.remotePort, __inbound.destinationAddress, __inbound.destinationPort),
      _loggingData ? sampledCounter1.increase() : sampledCounter0.increase()
    )
  )
)
.chain()
.handleMessage(
  msg => (
    loggingEnabled && _loggingData && (
      saveLoggingData(_loggingData, msg, __service?.name, __target)
    )
  )
)

))()