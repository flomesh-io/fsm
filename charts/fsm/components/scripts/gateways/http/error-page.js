((
  { config, isDebugEnabled } = pipy.solve('config.js'),

  errorPageCache = new algo.Cache(
    route => (
      (
        obj,
        status,
        errors = [],
        message = null,
        directory = null,
        errorPageTable = [],
        errorPage = route?.config?.ErrorPage || __domain?.ErrorPage || config?.Configs?.ErrorPage,
      ) => (
        errorPage && (errorPageTable = errorPage.map(
          errorConfig => (
            errors = (errorConfig?.Error || []).filter(
              e => e >= 100 && e <= 599
            ),
            (errors.length > 0) && errorConfig.Page && (
              status = (errorConfig.Status >= 100 && errorConfig.Status <= 599) && errorConfig.Status,
              obj = errorConfig.Page.toLowerCase(),
              (obj.startsWith('http://') || obj.startsWith('https://')) ? (
                status ||= 302,
                message = new Message({ status, headers: { Location: errorConfig.Page } })
              ) : (
                directory = errorConfig.Directory || __root
              ),
              (message || directory) && (
                {
                  status,
                  errors,
                  message,
                  directory,
                  path: errorConfig.Page,
                }
              )
            )
          )
        ).filter(e => e)),
        (errorPageTable.length > 0) ? (
          status => errorPageTable.find(ep => ep.errors.includes(status))
        ) : (
          null
        )
      )
    )()
  ),

) => (

pipy({
  _status: null,
  _request: null,
  _message: null,
  _directory: null,
  _errorPage: null,
  _matchStatus: null,
})

.import({
  __domain: 'route',
  __route: 'route',
  __root: 'web-server',
})

.pipeline()
.link('request')
.chain()
.link('response')

.pipeline('request')
.handleMessageStart(
  msg => _request = msg
)

.pipeline('response')
.onStart(
  () => void (
    _matchStatus = errorPageCache.get(__route)
  )
)
.branch(
  () => _matchStatus, (
    $=>$
    .handleMessageStart(
      msg => (
        (_errorPage = _matchStatus(+msg?.head?.status)) && (
          ((_message = _errorPage.message) || (_directory = _errorPage.directory)) && (
            _status = _errorPage.status || msg.head.status
          )
        )
      )
    )
    .branch(
      () => _message, (
        $=>$
        .replaceData()
        .replaceMessage(
          () => _message
        )
      ),
      () => _directory && (__root = _directory), (
        $=>$
        .replaceData()
        .replaceMessage(
          () => (
            new Message({ ..._request.head, method: 'GET', path: _errorPage.path })
          )
        )
        .use('server/web-server.js')
        .handleMessageStart(
          msg => msg.head && (
            msg.head.status = _status
          )
        )
      ), (
        $=>$
      )
    )
    .branch(
      isDebugEnabled, (
        $=>$
        .handleMessageStart(
          msg => _errorPage && (
            console.log('[error-page] config, message, directory, status, response:', _errorPage, _message, _directory, _status, msg)
          )
        )
      )
    )
  ), (
    $=>$
  )
)

))()