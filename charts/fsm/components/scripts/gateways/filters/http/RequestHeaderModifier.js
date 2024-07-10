export default function (config) {
  var $ctx

  var set = config.requestHeaderModifier.set
  var add = config.requestHeaderModifier.add
  var del = config.requestHeaderModifier.remove

  if (set) set = set.map(({ name, value }) => [name.toLowerCase(), value])
  if (add) add = add.map(({ name, value }) => [name.toLowerCase(), value, ',' + value])
  if (del) del = del.map(name => name.toLowerCase())

  return pipeline($=>$
    .onStart(c => void ($ctx = c))
    .handleMessageStart(
      function (req) {
        var headers = req.head.headers
        if (set) set.forEach(([k, v]) => headers[k] = v)
        if (add) add.forEach(([k, v, w]) => {
          var u = headers[k]
          if (u) {
            headers[k] = u + w
          } else {
            headers[k] = v
          }
        })
        if (del) del.forEach(k => delete headers[k])
      }
    )
    .pipeNext()
  )
}
