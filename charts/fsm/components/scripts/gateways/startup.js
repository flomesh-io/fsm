import resources from './resources.js'
import { isIdentical, log, makeFilters } from './utils.js'

var currentListeners = []
var dirtyGateways = []
var dirtyRouters = []
var dirtyBackends = []
var dirtyPolicies = []
var dirtyTimeout = null

export function startGateway(gateway) {
  gateway.spec.listeners.forEach(l => {
    try {
      var listenerKey = makeListener(gateway, l)
      if (!currentListeners.some(k => isIdentical(k, listenerKey))) {
        currentListeners.push(listenerKey)
      }
    } catch (err) {
      console.error(err)
    }
  })
}

function makeListener(gateway, listener) {
  var key = makeListenerKey(listener)
  var port = key[0]
  var wireProto = key[1]
  var routeModuleName
  var termTLS = false

  switch (listener.protocol) {
    case 'HTTP':
      routeModuleName = './modules/router-http.js'
      break
    case 'HTTPS':
      routeModuleName = './modules/router-http.js'
      termTLS = true
      break
    case 'TLS':
      switch (listener.tls?.mode) {
        case 'Terminate':
          routeModuleName = './modules/router-tcp.js'
          termTLS = true
          break
        case 'Passthrough':
          routeModuleName = './modules/router-tls.js'
          break
        default: throw `Listener: unknown TLS mode '${listener.tls?.mode}'`
      }
      break
    case 'TCP':
      routeModuleName = './modules/router-tcp.js'
      break
    case 'UDP':
      routeModuleName = './modules/router-udp.js'
      break
    default: throw `Listener: unknown protocol '${listener.protocol}'`
  }

  var routeResources = findRouteResources(gateway, listener)
  var routerKey = makeRouterKey(gateway, key)
  var pipelines = [pipy.import(routeModuleName).default(routerKey, listener, routeResources, gateway)]

  if (termTLS) {
    pipelines.unshift(
      pipy.import('./modules/terminate-tls.js').default(listener)
    )
  }

  if (listener.filters) {
    pipelines = [
      ...makeFilters(wireProto, listener.filters),
      ...pipelines,
    ]
  }

  var $ctx

  pipy.listen(port, wireProto, $=>$
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
    .onEnd(() => {
      log?.(`Inb #${$ctx.inbound.id} closed`)
    })
  )

  log?.(`Start listening on ${wireProto} port ${port}`)
  return key
}

function makeRouterKey(gateway, listenerKey) {
  return `${listenerKey.join(':')}:${gateway.metadata.name}`
}

function makeListenerKey(listener) {
  var address = listener.address || '0.0.0.0'
  var port = address.indexOf(':') >= 0 ? `[${address}]:${listener.port}` : `${address}:${listener.port}`
  var protocol = (listener.protocol === 'UDP' ? 'udp' : 'tcp')
  return [port, protocol]
}

function allRouteResources() {
  return [
    'HTTPRoute',
    'GRPCRoute',
    'TCPRoute',
    'TLSRoute',
    'UDPRoute',
  ].flatMap(kind => resources.list(kind))
}

function findRouteResources(gateway, listener) {
  var routeKinds = []
  switch (listener.protocol) {
    case 'HTTP':
    case 'HTTPS':
      routeKinds.push('HTTPRoute', 'GRPCRoute')
      break
    case 'TLS':
      switch (listener.tls?.mode) {
        case 'Terminate':
          routeKinds.push('TCPRoute')
          break
        case 'Passthrough':
          routeKinds.push('TLSRoute')
          break
      }
      break
    case 'TCP':
      routeKinds.push('TCPRoute')
      break
    case 'UDP':
      routeKinds.push('UDPRoute')
      break
  }

  return routeKinds.flatMap(kind => resources.list(kind)).filter(
    r => {
      var refs = r.spec?.parentRefs
      if (refs instanceof Array) {
        if (refs.some(
          r => {
            if (r.kind && r.kind !== 'Gateway') return false
            if (r.name !== gateway.metadata.name) return false
            if (r.sectionName === listener.name && listener.name) return true
            if (r.port == listener.port) return true
            return false
          }
        )) return true
      }
      return false
    }
  )
}

export function makeResourceWatcher(gatewayFilter) {
  if (!gatewayFilter) gatewayFilter = () => true

  return function onResourceChange(newResource, oldResource) {
    var res = newResource || oldResource
    var kind = res.kind
    var newName = newResource?.metadata?.name
    var oldName = oldResource?.metadata?.name
    switch (kind) {
      case 'Gateway':
        if (newName) addDirtyGateway(newName)
        if (oldName) addDirtyGateway(oldName)
        break
      case 'HTTPRoute':
      case 'GRPCRoute':
      case 'TCPRoute':
      case 'TLSRoute':
      case 'UDPRoute':
        addDirtyRouters(res.spec?.parentRefs)
        if (oldResource && res !== oldResource) {
          addDirtyRouters(oldResource.spec?.parentRefs)
        }
        break
      case 'BackendLBPolicy':
      case 'BackendTLSPolicy':
        addDirtyRoutersByPolicy(res.spec?.targetRefs)
        if (oldResource && res !== oldResource) {
          addDirtyRoutersByPolicy(oldResource.spec?.targetRefs)
        }
        break
      case 'HealthCheckPolicy':
        addDirtyPolicy(res)
        if (oldResource && res != oldResource) {
          addDirtyPolicy(oldResource)
        }
        break
      case 'Backend':
        if (newName) addDirtyBackend(newName)
        if (oldName) addDirtyBackend(oldName)
        break
    }
    if (dirtyTimeout) dirtyTimeout.cancel()
    dirtyTimeout = new Timeout(5)
    dirtyTimeout.wait().then(updateDirtyResources).catch(() => {})
  }

  function addDirtyGateway(name) {
    if (!dirtyGateways.includes(name)) {
      dirtyGateways.push(name)
      dirtyRouters = dirtyRouters.filter(
        ([kind, nm]) => (kind !== 'Gateway' || nm !== name)
      )
    }
  }

  function addDirtyRouters(refs) {
    if (refs instanceof Array) {
      refs.forEach(ref => {
        var key = [ref.kind, ref.name, ref.port, ref.sectionName]
        if (!dirtyRouters.some(k => isIdentical(k, key))) {
          dirtyRouters.push(key)
        }
      })
    }
  }

  function addDirtyBackend(name) {
    if (!dirtyBackends.includes(name)) {
      dirtyBackends.push(name)
    }
  }

  function addDirtyRoutersByPolicy(refs) {
    var dirtyBackendNames = Object.fromEntries(
      refs.filter(r => r.kind === 'Backend').map(r => [r.name, true])
    )
    allRouteResources().forEach(res => {
      var backendNames = res.spec?.rules?.flatMap?.(
        r => r.backendRefs?.map?.(r => r.name) || []
      )
      if (backendNames && backendNames.some(bn => bn in dirtyBackendNames)) {
        addDirtyRouters(res.spec.parentRefs)
      }
    })
  }

  function addDirtyPolicy(policy) {
    var kind = policy.kind
    policy.spec?.targetRefs?.forEach?.(
      ref => {
        var key = [kind, ref.kind, ref.name]
        if (!dirtyPolicies.some(k => isIdentical(k, key))) {
          dirtyPolicies.push(key)
        }
      }
    )
  }

  function updateDirtyResources() {
    var gateways = resources.list('Gateway')

    dirtyBackends.forEach(
      backendName => {
        if (resources.runUpdaters('Backend', backendName)) {
          log?.(`Updated backend ${backendName}`)
        }
      }
    )

    dirtyRouters.forEach(
      ([kind, name, port, sectionName]) => {
        if (kind !== 'Gateway') return
        var gw = gateways.find(gw => gw.metadata?.name === name)
        if (!gatewayFilter(gw)) return
        var l = gw?.spec?.listeners?.find?.(
          l => {
            if (l.name === sectionName && sectionName) return true
            if (l.port === port) return true
            return false
          }
        )
        if (!l) return
        var listenerKey = makeListenerKey(l)
        var routerKey = makeRouterKey(gw, listenerKey)
        var routeResources = findRouteResources(gw, l)
        if (resources.runUpdaters('Route', routerKey, l, routeResources)) {
          log?.(`Updated router ${routerKey}`)
        }
      }
    )

    dirtyPolicies.forEach(
      ([kind, refKind, refName]) => {
        if (resources.runUpdaters(kind, refName)) {
          log?.(`Updated ${kind} for ${refKind}/${refName}`)
        }
      }
    )

    if (dirtyGateways.length > 0) {
      var updatedListeners = []
      gateways.forEach(gw => {
        var name = gw.metadata?.name
        if (name && gatewayFilter(gw)) {
          var isUpdated = dirtyGateways.includes(name)
          gw.spec.listeners.forEach(l => {
            try {
              var listenerKey = isUpdated ? makeListener(gw, l) : makeListenerKey(l)
              if (!updatedListeners.some(k => isIdentical(k, listenerKey))) {
                updatedListeners.push(listenerKey)
              }
            } catch (err) {
              console.error(err)
            }
          })
        }
      })
      currentListeners.forEach(
        listenerKey => {
          if (!updatedListeners.some(k => isIdentical(k, listenerKey))) {
            var port = listenerKey[0]
            var protocol = listenerKey[1]
            pipy.listen(port, protocol, null)
            log?.(`Stop listening on ${protocol} port ${port}`)
          }
        }
      )
      currentListeners = updatedListeners
    }

    dirtyGateways = []
    dirtyRouters = []
    dirtyBackends = []
    dirtyPolicies = []
  }
}
