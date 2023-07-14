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
    metricRequestCount = new stats.Counter('request_count', ["host", "path", "method"]),
    metricResponseStatus = new stats.Counter('response_status', ["host", "path", "method", "status"]),
    metricResponseLatency = new stats.Histogram(
      'request_latency',
      new Array(26).fill().map((_,i) => Math.pow(1.5, i+1)|0).concat([Infinity]),
      ["host", "path", "method"],
    ),
) => pipy({
    _requestTime: 0,
    _reqHead: null,
  })

  .import({
    __route: 'main',
  })

  .pipeline()
    .handleMessageStart(
      (msg) => (
        _reqHead = msg.head,
        _requestTime = Date.now(),
        metricRequestCount.withLabels(_reqHead.headers.host, _reqHead.path, _reqHead.method).increase()
      )
    )
    .chain()
    .handleMessageStart(
      msg => (
        metricResponseLatency.withLabels(_reqHead.headers.host, _reqHead.path, _reqHead.method).observe(Date.now() - _requestTime),
        metricResponseStatus.withLabels(_reqHead.headers.host, _reqHead.path, _reqHead.method, msg.head.status || 200).increase()
      )
    )
)()