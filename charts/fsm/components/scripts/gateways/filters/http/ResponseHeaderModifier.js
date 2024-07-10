export default function (config) {
  var $ctx

  var set = config.responseHeaderModifier.set
  var add = config.responseHeaderModifier.add
  var del = config.responseHeaderModifier.remove

  if (set) set = set.map(({ name, value }) => [name.toLowerCase(), value])
  if (add) add = add.map(({ name, value }) => [name.toLowerCase(), value, ',' + value])
  if (del) del = del.map(name => name.toLowerCase())

  return pipeline($=>$
    .onStart(c => void ($ctx = c))
    .pipeNext()
    .handleMessageStart(
      function (res) {
        var headers = res.head.headers
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
  )
}
