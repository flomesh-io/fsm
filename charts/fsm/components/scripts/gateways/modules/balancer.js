import makeHTTP from './balancer-http.js'
import makeTCP from './balancer-tcp.js'
import makeUDP from './balancer-udp.js'
import makeDubbo from './balancer-dubbo.js'

export default function (protocol, backendRef, backendResource, options) {
  switch (protocol) {
    case 'http': return makeHTTP(backendRef, backendResource, options?.gateway, Boolean(options?.isHTTP2))
    case 'tcp': return makeTCP(backendRef, backendResource, options?.gateway)
    case 'udp': return makeUDP(backendRef, backendResource)
    case 'dubbo': return makeDubbo(backendRef, backendResource)
    default: throw `Invalid protocol '${protocol}' for balancer`
  }
}
