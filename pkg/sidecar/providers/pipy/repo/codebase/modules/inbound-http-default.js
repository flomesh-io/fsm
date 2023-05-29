pipy({
  _proxyError: false,
})

.import({
  __route: 'inbound-http-routing',
})

.pipeline()
.handleMessageStart(
  msg => (
    _proxyError = Boolean(msg?.head?.headers?.['x-forwarded-for'])
  )
)
.replaceData()
.branch(
  () => _proxyError, (
    $=>$.replaceMessage(
      new Message({ status: 502 }, 'Proxy Error')
    )
  ),

  () => !__route, (
    $=>$.replaceMessage(
      new Message({
          status: 403
        },
        'Access denied'
      )
    )
  ),

  (
    $=>$.replaceMessage(
      new Message({
          status: 404
        }, 'Not found'
      )
    )
  )
)