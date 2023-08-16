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
    {config, logLogger} = pipy.solve('config.js'),
  ) => pipy({
    _reqHead: null,
    _reqBody: '',
    _reqTime: 0,
    _reqSize: 0,
    _resHead: null,
    _resTime: 0,
    _resSize: 0,
    _resBody: '',
    _instanceName: os.env.HOSTNAME,

    _CONTENT_TYPES: {
      '': true,
      'text/plain': true,
      'application/json': true,
      'application/xml': true,
      'multipart/form-data': true,
    },
  })

  .pipeline()
    .handleMessageStart(
      msg => (
        _reqHead = msg.head,
        _reqBody = msg?.body?.toString?.() || '',
        _reqTime = Date.now()
      )
    )
    .handleData(data => _reqSize += data.size)
    .chain()
    .handleMessageStart(
      (msg, contentType) => (
        _resHead = msg.head,
        _resTime = Date.now(),
        contentType = (msg.head.headers && msg.head.headers['content-type']) || '',
        _resBody = Boolean(_CONTENT_TYPES[contentType]) ? msg?.body?.toString?.() || '' : ''
      )
    )
    .handleData(data => _resSize += data.size)
    .handleMessageEnd(
      () => (
        logLogger?.log?.({
          req: {
            ..._reqHead,
            body: _reqBody,
          },
          res: {
            ..._resHead,
            body: _resBody,
          },
          x_parameters: { aid: '', igid: '', pid: '' },
          instanceName: _instanceName,
          reqTime: _reqTime,
          resTime: _resTime,
          endTime: Date.now(),
          reqSize: _reqSize,
          resSize: _resSize,
          remoteAddr: __inbound?.remoteAddress,
          remotePort: __inbound?.remotePort,
          localAddr: __inbound?.localAddress,
          localPort: __inbound?.localPort,
        })
      )
    )

)()