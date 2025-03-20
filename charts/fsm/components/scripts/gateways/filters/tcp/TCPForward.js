export default function () {
  var $ctx

  return pipeline($=>$
    .onStart(c => { $ctx = c })
    .connect(() => $ctx.originalTarget)
  )
}
