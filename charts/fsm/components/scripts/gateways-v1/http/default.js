pipy({
  _proxyError: false,
})

.import({
  __route: 'route',
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
  (
    $=>$.replaceMessage(
      new Message({
          status: 404
        }, 'Not found'
      )
    )
  )
)