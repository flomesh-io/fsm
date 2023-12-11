((
  { config } = pipy.solve('config.js'),
  proxyTag = (config?.Configs?.ProxyTag?.DstHostHeader || 'proxy-tag').toLowerCase(),
  origHost = (config?.Configs?.ProxyTag?.SrcHostHeader || 'orig-host').toLowerCase(),
) => pipy()

.pipeline()
.handleMessageStart(
  msg => (
    msg?.head?.headers?.[proxyTag] ? ( // ingress
      msg.head.headers[origHost] = msg.head.headers.host,
      msg.head.headers.host = msg.head.headers[proxyTag]
    ) : msg?.head?.headers?.['fgw-target'] && ( // egress
      msg.head.headers[proxyTag] = msg.head.headers.host
    )
  )
)
.chain()

)()