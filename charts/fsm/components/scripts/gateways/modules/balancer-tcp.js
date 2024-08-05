import makeBackend from './backend.js'
import makeBackendTLS from './backend-tls.js'
import { log } from '../utils.js'

var $ctx
var $selection

export default function (backendRef, backendResource) {
  var name = backendResource.metadata.name
  var backend = makeBackend(name)
  var balancer = backend.balancer
  var tls = makeBackendTLS(backendRef, backendResource)

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
      'pass': (
        tls ? (
          $=>$.connectTLS({
            ...tls,
            onState: session => {
              if (session.error) {
                log?.(`Inb #${$ctx.inbound.id} tls error:`, session.error)
              }
            }
          }).to($=>$
            .connect(() => $selection.target.address).onEnd(() => $selection.free())
          )
        ) : (
          $=>$.connect(() => $selection.target.address).onEnd(() => $selection.free())
        )
      ),
      'deny': $=>$.replaceStreamStart(new StreamEnd),
    })
  )
}
