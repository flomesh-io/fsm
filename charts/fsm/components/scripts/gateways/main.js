#!/usr/bin/env -S pipy --args

import options from './options.js'
import resources from './resources.js'
import { log, logEnable, makeFilters } from './utils.js'

var opts = options(pipy.argv, {
  defaults: {
    '--config': '',
    '--debug': false,
  },
  shorthands: {
    '-c': '--config',
    '-d': '--debug',
  },
})

logEnable(opts['--debug'])
resources.init(opts['--config'], onResourceChange)

var $ctx

resources.list('Gateway').forEach(gw => {
  var gatewayName = gw.metadata?.name
  if (!gatewayName) return

  gw.spec.listeners.forEach(l => {
    var wireProto
    var routeKind
    var routeModuleName
    var termTLS = false

    switch (l.protocol) {
      case 'HTTP':
        wireProto = 'tcp'
        routeKind = 'HTTPRoute'
        routeModuleName = './modules/router-http.js'
        break
      case 'HTTPS':
        wireProto = 'tcp'
        routeKind = 'HTTPRoute'
        routeModuleName = './modules/router-http.js'
        termTLS = true
        break
      case 'TLS':
        wireProto = 'tcp'
        switch (l.tls?.mode) {
          case 'Terminate':
            routeKind = 'TCPRoute'
            routeModuleName = './modules/router-tcp.js'
            termTLS = true
            break
          case 'Passthrough':
            routeKind = 'TLSRoute'
            routeModuleName = './modules/router-tls.js'
            break
          default: throw `Listener: unknown TLS mode '${l.tls?.mode}'`
        }
        break
      case 'TCP':
        wireProto = 'tcp'
        routeKind = 'TCPRoute'
        routeModuleName = './modules/router-tcp.js'
        break
      case 'UDP':
        wireProto = 'udp'
        routeKind = 'UDPRoute'
        routeModuleName = './modules/router-udp.js'
        break
      default: throw `Listener: unknown protocol '${l.protocol}'`
    }

    var routeKinds = [routeKind]
    if (routeKind === 'HTTPRoute') routeKinds.push('GRPCRoute')

    var routeResources = routeKinds.flatMap(kind => resources.list(kind)).filter(
      r => {
        var refs = r.spec?.parentRefs
        if (refs instanceof Array) {
          if (refs.some(
            r => {
              if (r.kind && r.kind !== 'Gateway') return false
              if (r.name !== gw.metadata.name) return false
              if (r.sectionName === l.name && l.name) return true
              if (r.port == l.port) return true
              return false
            }
          )) return true
        }
        return false
      }
    )

    var routerKey = [gatewayName, l.address, l.port, l.protocol]
    var pipelines = [pipy.import(routeModuleName).default(routerKey, l, routeResources)]

    if (termTLS) {
      pipelines.unshift(
        pipy.import('./modules/terminate-tls.js').default(l)
      )
    }

    if (l.filters) {
      pipelines = [
        ...makeFilters(wireProto, l.filters),
        ...pipelines,
      ]
    }

    pipy.listen(l.port, wireProto, $=>$
      .onStart(i => {
        $ctx = {
          inbound: i,
          originalTarget: undefined,
          originalServerName: undefined,
          messageCount: 0,
          serverName: undefined,
          serverCert: null,
          clientCert: null,
          backendResource: null,
        }
        log?.(`Inb #${i.id} accepted on [${i.localAddress}]:${i.localPort} from [${i.remoteAddress}]:${i.remotePort}`)
        return new Data
      })
      .pipe(pipelines, () => $ctx)
    )

    log?.(`Listening ${l.protocol} on ${l.port}`)
  })
})

var dirtyRouters = {}
var dirtyBackends = []

function onResourceChange(newResource, oldResource) {
  var res = newResource || oldResource
  var kind = res.kind
  var name = res.metadata?.name
  switch (kind) {
    case 'Gateway':
      break
    case 'HTTPRoute':
    case 'TCPRoute':
    case 'TLSRoute':
    case 'UDPRoute':
      addDirtyRouters(res.spec?.parentRefs)
      if (oldResource && res !== oldResource) addDirtyRouters(oldResource.spec?.parentRefs)
      break
    case 'Backend':
      if (name) {
        dirtyBackends[name] = newResource
      }
      break
  }
}

function addDirtyRouters(refs) {
  if (refs instanceof Array) {
    refs.forEach(ref => {
      if (!dirtyRouters.some(r => isEqualListenerRef(r, ref))) {
        dirtyRouters.push(ref)
      }
    })
  }
}

function isEqualRef(a, b) {
  if (a.kind !== b.kind) return false
  if (a.name !== b.name) return false
  return true
}

function isEqualListenerRef(a, b) {
  if (!isEqualRef(a, b)) return false
  if (a.port !== b.port) return false
  if (a.sectionName !== b.sectionName) return false
  return true
}
