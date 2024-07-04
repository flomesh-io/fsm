export default function (config) {
  var $ctx

  var hostname = config.urlRewrite.hostname
  var rewriter = null

  switch (config.urlRewrite.path?.type) {
    case 'ReplaceFullPath':
      var path = config.urlRewrite.path.replaceFullPath
      rewriter = function () {
        return path
      }
      break
    case 'ReplacePrefixMatch':
      var prefix = config.urlRewrite.path.replacePrefixMatch
      rewriter = function (path) {
        if ($ctx.basePath) {
          return os.path.join(prefix, path.substring($ctx.basePath.length)) || '/'
        } else {
          return path
        }
      }
      break
  }

  return pipeline($=>$
    .onStart(c => void ($ctx = c))
    .handleMessageStart(
      function (req) {
        var head = req.head
        if (hostname) head.headers.host = hostname
        if (rewriter) head.path = rewriter(head.path)
      }
    )
    .pipeNext()
  )
}
