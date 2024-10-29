import resources from '../resources.js'
import makeBackendSelector from './backend-selector.js'
import makeBalancer from './balancer-tcp.js'
import { log } from '../utils.js'

var shutdown = pipeline($=>$.replaceStreamStart(new StreamEnd))

var $ctx
var $proto
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
    .onStart(c => void ($ctx = c))
    .detectProtocol(proto => void ($proto = proto))
    .pipe(
      () => {
        if ($proto !== undefined) {
          log?.(`Inb #${$ctx.inbound.id} protocol ${$proto || 'unknown'}`)
          return $proto === 'TLS' ? 'pass' : 'deny'
        }
      }, {
        'pass': ($=>$
          .handleTLSClientHello(router)
          .pipe(() => {
            if ($selection !== undefined) {
              return $selection ? $selection.target.pipeline : shutdown
            }
          })
        ),
        'deny': $=>$.replaceStreamStart(new StreamEnd),
      }
    )
  )
}

function makeRouter(listener, routeResources, gateway) {
  var hostFullnames = {}
  var hostPostfixes = []

  routeResources.forEach(r => {
    var hostnames = r.spec.hostnames || ['*']
    hostnames.forEach(name => {
      var selector = makeBackendSelector(
        'tcp', listener, r.spec.rules?.[0],
        function (backendRef, backendResource, filters) {
          var forwarder = backendResource ? makeBalancer(backendRef, backendResource, gateway) : shutdown
          return pipeline($=>$
            .pipe([...filters, forwarder], () => $ctx)
            .onEnd(() => $selection.free?.())
          )
        }
      )
      name = name.trim().toLowerCase()
      if (name.startsWith('*')) {
        hostPostfixes.push([name.substring(1), selector])
      } else {
        hostFullnames[name] = selector
      }
    })
  })

  hostPostfixes.sort((a, b) => b[0].length - a[0].length)

  return function (hello) {
    var sni = hello.serverNames[0] || ''
    var name = sni.toLowerCase()
    var selector = hostFullnames[name] || (
      hostPostfixes.find(
        ([postfix]) => name.endsWith(postfix)
      )?.[1]
    )
    $selection = selector?.() || null
    log?.(
      `Inb #${$ctx.inbound.id}`,
      `sni ${sni}`,
      `backend ${$selection?.target?.backendRef?.name}`
    )
  }
}
