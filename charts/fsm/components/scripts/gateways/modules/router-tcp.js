import resources from '../resources.js'
import makeBackendSelector from './backend-selector.js'
import makeBalancer from './balancer-tcp.js'
import { log } from '../utils.js'

var shutdown = pipeline($=>$.replaceStreamStart(new StreamEnd))

var $ctx
var $selection

export default function (routerKey, listener, routeResources, gateway) {
  var router = null

  function watch() {
    resources.setUpdater('Route', routerKey, update)
  }

  function update(listener, routeResources) {
    router = makeRouter(listener, routeResources, gateway)
    watch()
  }

  update(listener, routeResources)

  return pipeline($=>$
    .onStart(c => {
      $ctx = c
      router()
    })
    .pipe(() => $selection ? $selection.target.pipeline : shutdown)
  )
}

function makeRouter(listener, routeResources, gateway) {
  var selector = makeBackendSelector(
    'tcp', listener,
    routeResources[0]?.spec?.rules?.[0],
    function (backendRef, backendResource, filters) {
      var forwarder = backendResource ? makeBalancer(backendRef, backendResource, gateway) : shutdown
      return pipeline($=>$
        .pipe([...filters, forwarder], () => $ctx)
        .onEnd(() => $selection.free?.())
      )
    }
  )

  return function () {
    $selection = selector()
    log?.(
      `Inb #${$ctx.inbound.id}`,
      `backend ${$selection?.target?.backendRef?.name}`
    )
  }
}
