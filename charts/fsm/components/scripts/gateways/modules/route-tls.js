import makeBackendSelector from './backend-selector.js'
import makeForwarder from './forward-tcp.js'
import { log } from '../log.js'

var $ctx
var $proto
var $selection

export default function (config, listener, routeResources) {
  var shutdown = pipeline($=>$.replaceStreamStart(new StreamEnd))

  var hostFullnames = {}
  var hostPostfixes = []

  routeResources.forEach(r => {
    var hostnames = r.spec.hostnames || ['*']
    hostnames.forEach(name => {
      var selector = makeBackendSelector(
        config, 'tcp', r.spec.rules?.[0],
        function (backendRef, backendResource, filters) {
          var forwarder = backendResource ? makeForwarder(config, backendRef, backendResource) : shutdown
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

  function route(hello) {
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
          .handleTLSClientHello(route)
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
