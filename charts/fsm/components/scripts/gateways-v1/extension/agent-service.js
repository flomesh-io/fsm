((
) => pipy()

.pipeline()
.handleMessageStart(
  msg => (
    msg?.head?.headers?.['hy-agent-rsn'] && (
      msg.head.headers['orig-host'] = msg.head.headers.host,
      msg.head.headers.host = msg.head.headers['hy-agent-rsn']
    )
  )
)
.chain()

)()