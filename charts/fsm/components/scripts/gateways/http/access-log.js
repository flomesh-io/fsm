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

  formatFn = () => `${__inbound.remoteAddress} "${__consumer?.name || ''}" [${new Date(__requestTime).toString()}] ${Date.now() - __requestTime} ${_host || ''} "${_path}" ${__requestTail?.headSize} ${__requestTail?.bodySize} ${__responseHead?.status} ${__responseTail?.headSize || 0} ${__responseTail?.bodySize || 0} "${__requestHead?.headers?.['user-agent'] || ''}" "${_xff || ''}" "${__requestHead?.headers?.referer || ''}"`

) => pipy({
  _accessLog: null,
  _host: null,
  _path: null,
  _xff: null,
})

.import({
  __consumer: 'consumer',
  __requestHead: 'http',
  __requestTail: 'http',
  __requestTime: 'http',
  __responseHead: 'http',
  __responseTail: 'http',
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
        !__responseHead && (__responseHead = msg.head)
      )
    )
    .handleMessageEnd(
      msg => (
        !__responseTail && (__responseTail = msg.tail),
        _accessLog(formatFn())
      )
    )
  ), (
    $=>$
  )
)

)()