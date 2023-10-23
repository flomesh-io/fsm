((
  { config } = pipy.solve('config.js'),

  accessLogCache = new algo.Cache(
    route => (
      (
        al = route?.config?.AccessLog || __domain?.AccessLog || config?.Configs?.AccessLog
      ) => (
        al && new logging.TextLogger('access-log#' + al).toFile(al).log
      )
    )()
  ),

  formatFn = () => `${__inbound.remoteAddress} "${__consumer?.name || ''}" [${new Date(__request?.reqTime).toString()}] ${Date.now() - __request?.reqTime} ${_host || ''} "${_path}" ${__request?.tail?.headSize} ${__request?.tail?.bodySize} ${__response?.head?.status} ${__response?.tail?.headSize || 0} ${__response?.tail?.bodySize || 0} "${__request?.head?.headers?.['user-agent'] || ''}" "${_xff || ''}" "${__request?.head?.headers?.referer || ''}"`

) => pipy({
  _accessLog: null,
  _host: null,
  _path: null,
  _xff: null,
})

.import({
  __consumer: 'consumer',
  __request: 'http',
  __response: 'http',
  __domain: 'route',
  __route: 'route',
})

.pipeline()
.handleMessageStart(
  msg => (
    _host = msg?.head?.headers?.host,
    _path = msg?.head?.path,
    _xff = msg?.head?.headers?.['x-forwarded-for']
  )
)
.chain()
.branch(
  () => (_accessLog = accessLogCache.get(__route)), (
    $=>$
    .handleMessageStart(
      msg => (
        !__response && (__response = msg)
      )
    )
    .handleMessageEnd(
      msg => (
        !__response.tail && (__response.tail = msg.tail),
        _accessLog(formatFn())
      )
    )
  ), (
    $=>$
  )
)

)()