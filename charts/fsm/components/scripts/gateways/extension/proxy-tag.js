((
) => pipy()

.pipeline()
.handleMessageStart(
  msg => (
    msg?.head?.headers?.['proxy-tag'] ? ( // ingress
      msg.head.headers['orig-host'] = msg.head.headers.host,
      msg.head.headers.host = msg.head.headers['proxy-tag']
    ) : msg?.head?.headers?.['fgw-target'] && ( // egress
      msg.head.headers['proxy-tag'] = msg.head.headers.host
    )
  )
)
.chain()

)()
