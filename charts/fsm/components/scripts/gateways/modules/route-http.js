import makeBackendSelector from './backend-selector.js'
import makeForwarder from './forward-http.js'
import makeSessionPersistence from './session-persistence.js'
import { stringifyHTTPHeaders } from '../utils.js'
import { log } from '../log.js'

var $ctx
var $hostname
var $basePath
var $matchedRoute
var $matchedRule
var $selection

export default function (config, listener, routeResources) {
  var response404 = pipeline($=>$.replaceMessage(new Message({ status: 404 })))
  var response500 = pipeline($=>$.replaceMessage(new Message({ status: 500 })))

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

  function route(msg) {
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
        return rule.matches.map(m => [m, backendSelector, resource, rule])
      } else {
        return [[{}, backendSelector, resource, rule]]
      }
    })

    matches.sort((a, b) => {
      var ra = getMatchPriority(a[0])
      var rb = getMatchPriority(b[0])
      var i = ra.findIndex((r, i) => r !== rb[i])
      if (i < 0) return 0
      return ra[i] > rb[i] ? -1 : 1
    })

    function getMatchPriority(match) {
      var ranks = new Array(5)
      switch (match.path?.type) {
        case 'Exact':
          ranks[0] = 3
          ranks[1] = match.path.value
          break
        case 'PathPrefix':
          ranks[0] = 2
          ranks[1] = match.path.value.length
          break
        case 'RegularExpression':
          ranks[0] = 1
          ranks[1] = match.path.value.length
          break
        default:
          ranks[0] = 0
          ranks[1] = 0
          break
      }
      ranks[2] = Boolean(match.method)
      ranks[3] = match.headers?.length || 0
      ranks[4] = match.queryParams?.length || 0
      return ranks
    }

    switch (kind) {
      case 'HTTPRoute':
        matches = matches.map(([m, backendSelector, resource, rule], i) => {
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
          return [matchFunc, backendSelector]
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
          return [matchFunc, backendSelector]
        })
        break
      default: throw `route-http: unknown resource kind: '${kind}'`
    }
    return function (head) {
      var m = matches.find(([matchFunc]) => matchFunc(head))
      if (m) return m[1](head)
    }
  }

  function makeMethodMatcher(match) {
    if (match) {
      return method => method === match
    }
  }

  function makePathMatcher(match) {
    if (match) {
      var type = match.type || 'Exact'
      var value = match.value
      switch (type) {
        case 'Exact':
          var patterns = new algo.URLRouter({ [value]: true })
          return path => patterns.find(path)
        case 'PathPrefix':
          var base = (value.endsWith('/') ? value.substring(0, value.length - 1) : value) || '/'
          var patterns = new algo.URLRouter({ [base]: true, [base + '/*']: true })
          return path => {
            if (patterns.find(path)) {
              $basePath = base
              return true
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
    var selector = makeBackendSelector(
      config, 'http', listener, rule,
      function (backendRef, backendResource, filters) {
        if (!backendResource) return response500
        var forwarder = makeForwarder(config, backendRef, backendResource, isHTTP2)
        if (sessionPersistence) {
          var preserveSession = sessionPersistence.preserve
          return pipeline($=>$
            .pipe([...filters, forwarder], () => $ctx)
            .handleMessageStart(
              msg => preserveSession(msg.head, $selection?.target?.backendRef?.name)
            )
            .onEnd(() => $selection.free?.())
          )
        } else {
          return pipeline($=>$
            .pipe([...filters, forwarder], () => $ctx)
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

  return pipeline($=>$
    .onStart(c => void ($ctx = c))
    .demuxHTTP().to($=>$
      .handleMessageStart(
        function (msg) {
          route(msg)
          $ctx = {
            parent: $ctx,
            id: ++$ctx.messageCount,
            host: $hostname,
            path: msg.head.path,
            head: msg.head,
            headTime: Date.now(),
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
            target: '',
            retries: [],
          }
        }
      )
      .handleMessageEnd(
        function (msg) {
          $ctx.tail = msg.tail
          $ctx.tailTime = Date.now()
        }
      )
      .pipe(() => $selection ? $selection.target.pipeline : response404)
      .handleMessageStart(
        function (msg) {
          var r = $ctx.response
          r.head = msg.head
          r.headTime = Date.now()
        }
      )
      .handleMessageEnd(
        function (msg) {
          var r = $ctx.response
          r.tail = msg.tail
          r.tailTime = Date.now()
        }
      )
    )
  )
}
