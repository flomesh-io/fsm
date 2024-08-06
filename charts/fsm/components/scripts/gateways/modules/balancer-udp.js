import makeBackend from './backend.js'
import { log } from '../utils.js'

var $ctx
var $selection

export default function (backendRef, backendResource) {
  var name = backendResource.metadata.name
  var backend = makeBackend(name)
  var balancer = backend.balancer

  var isHealthy = (target) => true

  return pipeline($=>$
    .onStart(c => {
      $ctx = c
      $selection = balancer.allocate(null, isHealthy)
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
