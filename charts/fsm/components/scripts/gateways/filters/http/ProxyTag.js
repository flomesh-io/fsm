export default function (config) {
  proxyTag = (config.proxyTag?.dstHostHeader || 'proxy-tag').toLowerCase()
  origHost = (config.proxyTag?.srcHostHeader || 'orig-host').toLowerCase()

  return pipeline($=>$
    .handleMessageStart(
      msg => {
        var headers = msg.head.headers
        var tag = headers[proxyTag]
        if (tag) {
          headers[origHost] = headers.host
          headers.host = tag
        } else if (headers['fgw-target']) {
          headers[proxyTag] = headers.host
        }
      }
    )
    .pipeNext()
  )
}
