import makeBackendSelector from './backend-selector.js'
import makeForwarder from './forward-tcp.js'
import { log } from '../log.js'

var $ctx
var $selection

export default function (config, listener, routeResources) {
  var shutdown = pipeline($=>$.replaceStreamStart(new StreamEnd))

  var selector = makeBackendSelector(
    config, 'tcp', listener,
    routeResources[0]?.spec?.rules?.[0],
    function (backendRef, backendResource, filters) {
      var forwarder = backendResource ? makeForwarder(config, backendRef, backendResource) : shutdown
      return pipeline($=>$
        .pipe([...filters, forwarder], () => $ctx)
        .onEnd(() => $selection.free?.())
      )
    }
  )

  function route() {
    $selection = selector()
    log?.(
      `Inb #${$ctx.inbound.id}`,
      `backend ${$selection?.target?.backendRef?.name}`
    )
  }

  return pipeline($=>$
    .onStart(c => {
      $ctx = c
      route()
    })
    .pipe(() => $selection ? $selection.target.pipeline : shutdown)
  )
}
