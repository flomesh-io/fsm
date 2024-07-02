import { log } from '../log.js'

var $ctx
var $selection

export default function (config, backendRef, backendResource) {
  var targets = backendResource ? backendResource.spec.targets.map(t => {
    var port = t.port || backendRef.port
    var address = `${t.address}:${port}`
    var weight = t.weight
    return { address, weight }
  }) : []

  var loadBalancer = new algo.LoadBalancer(
    targets, {
      key: t => t.address,
      weight: t => t.weight,
    }
  )

  var isHealthy = (target) => true

  return pipeline($=>$
    .onStart(c => {
      $ctx = c
      $selection = loadBalancer.allocate(null, isHealthy)
      log?.(
        `Inb #${$ctx.inbound.id}`,
        `target ${$selection?.target?.address}`
      )
    })
    .pipe(() => $selection ? 'pass' : 'deny', {
      'pass': $=>$.connect(() => $selection.target.address, { protocol: 'udp' }).onEnd(() => $selection.free()),
      'deny': $=>$.replaceStreamStart(new StreamEnd),
    })
  )
}
