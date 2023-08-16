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

  extTypes = ((mime = JSON.decode(pipy.load('files/mime.json'))) => (
    Object.fromEntries(
      Object.keys(mime || {}).map(
        exts => (
          exts.split(' ').filter(o => o).map(
            ext => ([ext, mime[exts]])
          )
        )
      ).flat()
    )
  ))(),

  matchContentType = ext => (
    extTypes?.[ext] || 'text/html'
  ),

) => pipy({
  _pos: 0,
  _file: null,
  _data: null,
  _extName: null,
  _filepath: null,
  _message: undefined,
})

.export('web-server', {
  __root: null,
})

.pipeline()
.replaceMessage(
  msg => (
    msg?.head?.path && (msg.head.path.indexOf('/../') < 0) && (
      __root?.startsWith('/') ? (
        _filepath = __root + msg.head.path,
        msg.head.path.endsWith('/') ? (
          _filepath = _filepath + 'index.html'
        ) : (
          (msg.head.path.lastIndexOf('/') > msg.head.path.lastIndexOf('.')) && (
            _filepath = _filepath + '/index.html'
          )
        ),
        _filepath = _filepath.replace('//', '/'),
        ((_pos = _filepath.lastIndexOf('.')) > 0) && (
          _extName = _filepath.substring(_pos + 1)
        ),
        os.stat(_filepath)?.isFile?.() && (_data = os.readFile(_filepath)) && (
          _message = new Message({ status: 200, headers: {'content-type': matchContentType(_extName)}}, _data)
        )
      ) : (
        __root && (
          _filepath = 'static/' + __root + msg.head.path,
          msg.head.path.endsWith('/') ? (
            _filepath = _filepath + 'index.html'
          ) : (
            (msg.head.path.lastIndexOf('/') > msg.head.path.lastIndexOf('.')) && (
              _filepath = _filepath + '/index.html'
            )
          ),
          _filepath = _filepath.replace('//', '/'),
          ((_pos = _filepath.lastIndexOf('.')) > 0) && (
            _extName = _filepath.substring(_pos + 1)
          ),
          (_file = http.File.from(_filepath)) && (
            (_message = _file.toMessage(msg.head?.headers?.['accept-encoding'])) && (
              _message.head.headers['content-type'] = matchContentType(_extName)
            )
          )
        )
      )
    ),
    _message ? (
      _message
    ) : (
      new Message({status: 404}, 'Not Found')
    )
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleMessage(
      msg => (
        console.log('[web-server] _filepath, _extName, status, content-type:', _filepath, _extName, msg?.head?.status, msg?.head?.headers?.['content-type'])
      )
    )
  )
)

)()