((
  { config, isDebugEnabled } = pipy.solve('config.js'),

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

  defaultMimeTypeCache = new algo.Cache(
    route => (
      route?.config?.DefaultMimeType ? (
        route.config.DefaultMimeType
      ) : (
        __domain?.DefaultMimeType ? (
          __domain.DefaultMimeType
        ) : (
          config?.Configs?.DefaultMimeType
        )
      )
    )
  ),

  matchContentType = ext => (
    extTypes?.[ext] || defaultMimeTypeCache.get(__route) || 'application/octet-stream'
  ),

  checkFileMode = filepath => (
    (
      s = os.stat(filepath)
    ) => (
      s ? (
        ((s.mode & 16384) === 16384) ? 1 : 0
      ) : -1
    )
  )(),

  getExt = url => (
    (
      dot = url.lastIndexOf('.'),
      slash = url.lastIndexOf('/'),
    ) => (
      (dot > slash) ? (
        url.substring(dot + 1)
      ) : ''
    )
  )(),

  dirCache = new algo.Cache(
    dir => (
      dir.startsWith('/') ? (
        new http.Directory(dir, { fs: true, index: (__route.config?.Index || ['index.html']) })
      ) : (
        new http.Directory('/static/' + dir, { fs: false, index: (__route.config?.Index || ['index.html']) })
      )
    ),
    null,
    { ttl: 3600 }
  ),

  tryFilesCache = new algo.Cache(
    route => (
      (
        uriCache = new algo.Cache(
          uri => route?.config?.TryFiles && (
            route.config.TryFiles.map(
              f => (
                (
                  e = f.split('/')
                ) => (
                  e.map(
                    i => i.replace('$uri', uri)
                  ).join('/').replace('//', '/')
                )
              )()
            )
          ),
          null,
          { ttl: 3600 }
        ),
      ) => (
        uri => uriCache.get(uri)
      )
    )()
  ),

  makeMessage = (uri, msg) => (
    uri.startsWith('=') ? (
      (
        status = uri.substring(1)
      ) => (
        (status > 0) ? (
          new Message({ status })
        ) : null
      )
    )() : (
      (_dir = dirCache.get(__root)) ? (
        msg.head.path = _filepath = uri,
        _extName = getExt(uri),
        _dir.serve(msg)
      ) : null
    )
  ),

) => pipy({
  _uri: null,
  _dir: null,
  _path: null,
  _message: null,
  _extName: null,
  _filepath: null,
  _tryFiles: null,
})

.export('web-server', {
  __root: null,
})

.import({
  __domain: 'route',
  __route: 'route',
})

.pipeline()
.replaceMessage(
  msg => (
    (_path = msg?.head?.path) && (
      _uri = _path.split('?')[0]
    ),
    _uri && (_uri.indexOf('/../') < 0) && (
      _tryFiles = tryFilesCache.get(__route)?.(_uri),
      (_tryFiles || [_uri]).find(
        tf => (
          (_message = makeMessage(tf, msg)) && _extName && (
            _message.head.headers['content-type'] = matchContentType(_extName)
          ),
          _message
        )
      )
    ),
    _message || new Message({ status: 404 }, 'Not Found')
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleMessageStart(
      msg => (
        console.log('[web-server] _path, _filepath, _extName, status, content-type:', _path, _filepath, _extName, msg?.head?.status, msg?.head?.headers?.['content-type'])
      )
    )
  )
)

)()
