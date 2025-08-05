import resources from '../resources.js'
import makeBackendSelector from './backend-selector.js'
import makeBalancer from './balancer.js'
import makeSessionPersistence from './session-persistence.js'
import { log, stringifyHTTPHeaders } from '../utils.js'

var USE_FAST_MATCH = true

var response404 = pipeline($=>$.replaceMessage(new Message({ status: 404 })))
var response500 = pipeline($=>$.replaceMessage(new Message({ status: 500 })))

var $ctx
var $hostname
var $basePath
var $matchedRoute
var $matchedRule
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

  var handleRequest = pipeline($=>$
    .handleMessageStart(
      function (msg) {
        router(msg)
        $ctx = {
          parent: $ctx,
          id: ++$ctx.messageCount,
          host: $hostname,
          path: msg.head.path,
          head: msg.head,
          headTime: pipy.performance.now(),
          tail: null,
          tailTime: 0,
          sendTime: 0,
          response: {
            head: null,
            headTime: 0,
            tail: null,
            tailTime: 0,
          },
          basePath: $basePath,
          routeResource: $matchedRoute,
          routeRule: $matchedRule,
          backendResource: $selection?.target?.backendResource,
          backend: null,
          consumer: '',
          target: '',
          retries: [],
        }
      }
    )
    .handleMessageEnd(
      function (msg) {
        $ctx.tail = msg.tail
        $ctx.tailTime = pipy.performance.now()
      }
    )
    .pipe(() => $selection ? $selection.target.pipeline : response404)
  )

  var handleStream = pipeline($=>$
    .demuxHTTP().to(handleRequest)
  )

  return pipeline($=>$
    .onStart(c => void ($ctx = c))
    .pipe(evt => evt instanceof MessageStart ? handleRequest : handleStream)
  )
}

function makeRouter(listener, routeResources, gateway) {
  var hostFullnames = {}
  var hostPostfixes = {}

  routeResources.forEach(r => {
    var hostnames = r.spec.hostnames || ['*']
    hostnames.forEach(name => {
      name = name.trim().toLowerCase()
      if (name.startsWith('*')) {
        (hostPostfixes[name.substring(1)] ??= []).push(r)
      } else {
        (hostFullnames[name] ??= []).push(r)
      }
    })
  })

  hostFullnames = Object.fromEntries(
    Object.entries(hostFullnames).map(
      ([k, v]) => [k, makeRuleSelector(v)]
    )
  )

  hostPostfixes = Object.entries(hostPostfixes)
    .sort((a, b) => b[0].length - a[0].length)
    .map(([k, v]) => [k, makeRuleSelector(v)])

  return function (msg) {
    var head = msg.head
    var host = head.headers.host
    if (host) {
      host = host.toLowerCase()
      var i = host.lastIndexOf(':')
      if (i >= 0) host = host.substring(0, i)
      var selector = hostFullnames[host] || (
        hostPostfixes.find(
          ([postfix]) => host.endsWith(postfix)
        )?.[1]
      )
      if (selector) $selection = selector(head)
    }
    $hostname = host
    log?.(
      `Inb #${$ctx.inbound.id} Req #${$ctx.messageCount+1}`, head.method, head.path,
      `backend ${$selection?.target?.backendRef?.name}`,
      `headers ${stringifyHTTPHeaders(head.headers)}`,
    )
  }

  function makeRuleSelector(routeResources) {
    var kind = routeResources[0].kind
    var rules = routeResources.flatMap(resource => resource.spec.rules.map(
      r => [r, makeBackendSelectorForRule(r, kind === 'GRPCRoute'), resource]
    ))

    var matches = rules.flatMap(([rule, backendSelector, resource]) => {
      if (rule.matches) {
        return rule.matches.map(m => [m, backendSelector, resource, rule, ''])
      } else {
        return [[{}, backendSelector, resource, rule, '']]
      }
    })

    matches.forEach((a, i) => {
      a[4] = getMatchPriority(a[0], i)
    })

    matches.sort((a, b) => {
      return a[4] > b[4] ? -1 : 1
    })

    function getMatchPriority(match, index) {
      var ranks = new Array(5)
      switch (match.path?.type) {
        case 'Exact':
          ranks[0] = '3'
          ranks[1] = '000'
          break
        case 'PathPrefix':
          ranks[0] = '2'
          ranks[1] = match.path.value.length.toString().padStart(3, '0')
          break
        case 'RegularExpression':
          ranks[0] = '1'
          ranks[1] = match.path.value.length.toString().padStart(3, '0')
          break
        default:
          ranks[0] = '0'
          ranks[1] = '000'
          break
      }
      ranks[2] = (match.method ? '1' : '0')
      ranks[3] = (match.headers?.length || 0).toString().padStart(3, '0')
      ranks[4] = (match.queryParams?.length || 0).toString().padStart(3, '0')
      ranks[5] = index.toString().padStart(6, '0')
      return ranks.join('/')
    }

    switch (kind) {
      case 'HTTPRoute':
        matches = matches.map(([m, backendSelector, resource, rule]) => {
          var matchMethod = makeMethodMatcher(m.method)
          var matchPath = makePathMatcher(m.path)
          var matchHeaders = makeObjectMatcher(m.headers)
          var matchParams = makeObjectMatcher(m.queryParams)
          var matchFunc = function (head) {
            if (matchMethod && !matchMethod(head.method)) return false
            if (matchPath && !matchPath(head.path)) return false
            if (matchHeaders && !matchHeaders(head.headers)) return false
            if (matchParams && !matchParams(new URL(head.path).searchParams.toObject())) return false
            $matchedRoute = resource
            $matchedRule = rule
            return true
          }
          return [matchFunc, backendSelector, m]
        })
        break
      case 'GRPCRoute':
        matches = matches.map(([m, backendSelector]) => {
          var matchMethod = makeGRPCMethodMatcher(m.method)
          var matchHeaders = makeObjectMatcher(m.headers)
          var matchFunc = function (head) {
            if (matchMethod && !matchMethod(head.path)) return false
            if (matchHeaders && !matchHeaders(head.headers)) return false
            return true
          }
          return [matchFunc, backendSelector, m]
        })
        break
      default: throw `route-http: unknown resource kind: '${kind}'`
    }

    var pathMatches = {}
    var methodMatches = {}
    var slowMatches = []

    if (kind === 'HTTPRoute') {
      matches.forEach(match => {
        var m = match[2]
        if (m.path && (m.path.type === 'Exact' || m.path.type === 'PathPrefix')) {
          var path = m.path.value
          if (m.path.type === 'PathPrefix') path = os.path.join(path, '*')
          var methods = (pathMatches[path] ??= {})
          var checks = (methods[m.method || '*'] ??= [])
          checks.push(match)
        } else if (m.path && m.path.type === 'PathPrefix') {
          var path = m.path.value
          if (m.path.type === 'PathPrefix') path = os.path.join(path, '*')
          var methods = (pathMatches[path] ??= {})
          var checks = (methods[m.method || '*'] ??= [])
          checks.push(match)
        } else if (m.method) {
          var checks = (methodMatches[m.method] ??= [])
          checks.push(match)
        } else {
          slowMatches.push(match)
        }
      })
    } else {
      slowMatches = matches
    }

    pathMatches = new algo.URLRouter(pathMatches)

    if (USE_FAST_MATCH && kind === 'HTTPRoute') {
      return function (head) {
        var pm = pathMatches.find(head.path)
        if (pm) {
          var checks = pm[head.method] || pm['*']
          if (checks) {
            var m = checks.find(([matchFunc]) => matchFunc(head))
            if (m) return m[1](head)
          }
        }

        var checks = methodMatches[head.method]
        if (checks) {
          var m = checks.find(([matchFunc]) => matchFunc(head))
          if (m) return m[1](head)
        }

        var m = slowMatches.find(([matchFunc]) => matchFunc(head))
        if (m) return m[1](head)
      }

    } else {
      return function (head) {
        var m = matches.find(([matchFunc]) => matchFunc(head))
        if (m) return m[1](head)
      }
    }
  }

  function makeMethodMatcher(match) {
    if (USE_FAST_MATCH) {
      return null
    } else {
      if (match) {
        return method => method === match
      }
    }
  }

  function makePathMatcher(match) {
    if (match) {
      var type = match.type || 'Exact'
      var value = match.value
      switch (type) {
        case 'Exact':
          if (USE_FAST_MATCH) {
            return null
          } else {
            var patterns = new algo.URLRouter({ [value]: true })
            return path => patterns.find(path)
          }
        case 'PathPrefix':
          var base = (value.endsWith('/') ? value.substring(0, value.length - 1) : value) || '/'
          if (USE_FAST_MATCH) {
            return () => {
              $basePath = base
              return true
            }
          } else {
            var patterns = new algo.URLRouter({ [base]: true, [base + '/*']: true })
            return path => {
              if (patterns.find(path)) {
                $basePath = base
                return true
              }
            }
          }
        case 'RegularExpression':
          var re = new RegExp(value)
          return path => re.test(path)
        default: return () => false
      }
    }
  }

  function makeObjectMatcher(items) {
    if (items instanceof Array && items.length > 0) {
      var exact = items.filter(i => i.type === 'Exact' || !i.type).map(i => [i.name.toLowerCase(), i.value])
      var regex = items.filter(i => i.type === 'RegularExpression').map(i => [i.name.toLowerCase(), new RegExp(i.value)])
      return (obj) => (
        exact.every(([k, v]) => v === (obj[k] || '')) &&
        regex.every(([k, v]) => v.test(obj[k] || ''))
      )
    }
  }

  function makeGRPCMethodMatcher(match) {
    if (match) {
      var service = match.service
      var method = match.method
      var checkFunc
      switch (match.type || 'Exact') {
        case 'Exact':
          checkFunc = (s, m) => (!service || service === s) && (!method || method === m)
          break
        case 'RegularExpression':
          var reService = service && new RegExp(service)
          var reMethod = method && new RegExp(method)
          checkFunc = (s, m) => (!reService || reService.test(s)) && (!reMethod || reMethod.test(m))
          break
        default: return () => false
      }
      return path => {
        if (!path.startsWith('/')) return false
        var s = path.split('/')
        var service = s[1]
        var method = s[2]
        if (!service || !method) return false
        return checkFunc(service, method)
      }
    }
  }

  function makeBackendSelectorForRule(rule, isHTTP2) {
    var sessionPersistenceConfig = rule.sessionPersistence
    var sessionPersistence = sessionPersistenceConfig && makeSessionPersistence(sessionPersistenceConfig)
    var timeoutConfig = rule.timeouts
    var timeoutFrontendPipeline = null
    var timeoutBackendPipeline = null
    var retryConfig = rule.retry
    var retryPipeline = null
    var retryCodes = {}

    if (timeoutConfig) {
      var makeTimeoutPipeline = function (timeout) {
        var $branch
        return pipeline($=>$
          .forkRace(['forward', 'timeout']).to($=>$
            .onStart(b => { $branch = b })
            .pipe(() => $branch, {
              'timeout': ($=>$.wait(() => new Timeout(timeout).wait()).replaceData().replaceMessage(new Message({ status: 504 }))),
              'forward': ($=>$.pipeNext())
            })
          )
        )
      }
      var timeoutFrontend = Number.parseFloat(timeoutConfig.request) || Number.POSITIVE_INFINITY
      var timeoutBackend = Number.parseFloat(timeoutConfig.backendRequest) || Number.POSITIVE_INFINITY
      if (timeoutFrontend > 0 && Number.isFinite(timeoutFrontend)) timeoutFrontendPipeline = makeTimeoutPipeline(timeoutFrontend)
      if (timeoutBackend > 0 && Number.isFinite(timeoutBackend)) timeoutBackendPipeline = makeTimeoutPipeline(timeoutBackend)
    }

    if (retryConfig) {
      (retryConfig.codes || ['5xx']).forEach(code => {
        if (code.toString().substring(1) === 'xx') {
          var base = Number.parseInt(code.toString().charAt(0) + '00')
          new Array(100).fill().forEach((_, i) => retryCodes[base + i] = true)
        } else {
          retryCodes[Number.parseInt(code)] = true
        }
      })

      var $retryCounter = 0
      retryPipeline = pipeline($=>$
        .repeat(() => {
          if ($retryCounter > 0) {
            return new Timeout(retryConfig.backoff * Math.pow(2, $retryCounter - 1)).wait().then(true)
          } else {
            return false
          }
        }).to($=>$
          .pipeNext()
          .pipe(evt => {
            if (evt instanceof MessageStart) {
              var needRetry = (evt.head.status in retryCodes)
              var wasRetry = ($retryCounter > 0)
              if (wasRetry) {
                $ctx.retries.push({
                  target: $ctx.target,
                  succeeded: !needRetry,
                })
              }
              if (needRetry) {
                if (++$retryCounter < retryConfig.attempts) {
                  log?.(`Inb #${$ctx.parent.inbound.id} Req #${$ctx.id} retry ${$retryCounter} status ${evt.head.status}`)
                  return 'retry'
                } else {
                  log?.(`Inb #${$ctx.parent.inbound.id} Req #${$ctx.id} retry ${$retryCounter} status ${evt.head.status} gave up`)
                  $retryCounter = 0
                  return 'conclude'
                }
              } else {
                return 'conclude'
              }
            }
          }, {
            'retry': $=>$.replaceData().replaceMessage(new StreamEnd),
            'conclude': $=>$,
          })
        )
      )
    }

    var selector = makeBackendSelector(
      'http', listener, rule,

      function (backendRef, backendResource, filters, protocol) {
        if (!backendResource && filters.length === 0) return response500
        var forwarder = backendResource ? [makeBalancer(protocol, backendRef, backendResource, { gateway, isHTTP2 })] : []

        if (protocol === 'http') {
          if (retryPipeline) forwarder.unshift(retryPipeline)
          if (timeoutBackendPipeline) forwarder.unshift(timeoutBackendPipeline)
        }

        var filterChain = [...filters, ...forwarder]
        if (timeoutFrontendPipeline) filterChain.unshift(timeoutFrontendPipeline)

        if (sessionPersistence) {
          var preserveSession = sessionPersistence.preserve
          return pipeline($=>$
            .pipe(filterChain, () => $ctx)
            .handleMessageStart(
              msg => preserveSession(msg.head, $selection?.target?.backendRef?.name)
            )
            .onEnd(() => $selection.free?.())
          )
        } else {
          return pipeline($=>$
            .pipe(filterChain, () => $ctx)
            .onEnd(() => $selection.free?.())
          )
        }
      }
    )

    if (sessionPersistence) {
      var restoreSession = sessionPersistence.restore
      return (head) => selector(restoreSession(head))
    }

    return () => selector()
  }
}
