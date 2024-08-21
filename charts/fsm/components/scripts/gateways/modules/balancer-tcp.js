import makeBackend from './backend.js'
import makeBackendTLS from './backend-tls.js'
import { log } from '../utils.js'

var $ctx
var $session
var $conn

export default function (backendRef, backendResource) {
  var name = backendResource.metadata.name
  var backend = makeBackend(name)
  var balancer = backend.balancer
  var tls = makeBackendTLS(backendRef, backendResource)

  var isHealthy = (target) => true

  return pipeline($=>$
    .onStart(c => {
      $ctx = c
      $session = balancer.allocate(null, isHealthy)
      $conn = { protocol: 'tcp', target: $session?.target }
      log?.(
        `Inb #${$ctx.inbound.id}`,
        `target ${$session?.target?.address}`
      )
    })
    .pipe(() => $session ? 'pass' : 'deny', {
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
            .pipe(backend.connect, () => $conn)
            .onEnd(() => $session.free())
          )
        ) : ($=>$
          .pipe(backend.connect, () => $conn)
          .onEnd(() => $session.free())
        )
      ),
      'deny': $=>$.replaceStreamStart(new StreamEnd),
    })
  )
}
