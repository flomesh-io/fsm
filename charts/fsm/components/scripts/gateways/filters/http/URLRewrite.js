export default function (config) {
  var $ctx

  return pipeline($=>$
    .onStart(c => void ($ctx = c))
    .pipeNext()
  )
}
