import resources from '../resources.js'
import makeBackendSelector from './backend-selector.js'
import makeBalancer from './balancer.js'
import { log } from '../utils.js'

var response404 = pipeline($=>$.replaceMessage(new Message({ status: 404 })))
var response500 = pipeline($=>$.replaceMessage(new Message({ status: 500 })))

var $ctx
var $matchedRoute
var $matchedRule
var $selection

export default function (routerKey, listener, routeResources) {
  var router = null

  function watch() {
    resources.setUpdater('Route', routerKey, update)
  }

  function update(listener, routeResources) {
    router = makeRouter(listener, routeResources)
    watch()
  }

  update(listener, routeResources)

  var handleRequest = pipeline($=>$
    .handleMessageStart(
      function (msg) {
        $ctx = {
          parent: $ctx,
          id: ++$ctx.messageCount,
          head: msg.head,
          headTime: Date.now(),
          tailTime: 0,
          sendTime: 0,
          body: null,
          response: {
            head: null,
            headTime: 0,
            tailTime: 0,
          },
          routeResource: null,
          routeRule: null,
          backendResource: null,
          backend: null,
        }
      }
    )
    .replaceMessage(
      function (msg) {
        var head = msg.head
        var body = $ctx.body = Hessian.decode(msg.body)
        router(head, body)
        if ($selection) {
          $ctx.routeResource = $matchedRoute
          $ctx.routeRule = $matchedRule
          $ctx.backendResource = $selection.target?.backendResource
        }
        return msg
      }
    )
    .handleMessageEnd(
      function (msg) {
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
      function () {
        var r = $ctx.response
        r.tailTime = Date.now()
      }
    )
  )

  var handleStream = pipeline($=>$
    .decodeDubbo()
    .demuxQueue().to(handleRequest)
    .encodeDubbo()
  )

  return pipeline($=>$
    .onStart(c => void ($ctx = c))
    .pipe(evt => evt instanceof MessageStart ? handleRequest : handleStream)
  )
}

function makeRouter(listener, routeResources) {
  var cache = new algo.Cache({ ttl: 3600 })
  var selector = makeRuleSelector(routeResources)

  return function (head, body) {
    var key = body.slice(1,5).join(' ')
    var val = cache.get(key)
    if (val) {
      $selection = val
    } else {
      cache.set(key, $selection = selector(body))
    }
    log?.(
      `Inb #${$ctx.parent.inbound.id} Req #${$ctx.parent.messageCount+1}`, head.requestID,
      `backend ${$selection?.target?.backendRef?.name}`,
    )
  }

  function makeRuleSelector(routeResources) {
    var rules = routeResources.flatMap(resource => resource.spec.rules.map(
      r => [r, makeBackendSelectorForRule(r), resource]
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
      switch (match.type) {
        case 'Exact':
          ranks[0] = 3
          ranks[1] = match.service || ''
          ranks[2] = match.version || ''
          ranks[3] = match.method || ''
          ranks[4] = match.signature || ''
          break
        case 'RegularExpression':
          ranks[0] = 1
          ranks[1] = match.service?.length || 0
          ranks[2] = match.version?.length || 0
          ranks[3] = match.method?.length || 0
          ranks[4] = match.signature?.length || 0
          break
        default:
          ranks[0] = 0
          ranks[1] = 0
          ranks[2] = 0
          ranks[3] = 0
          ranks[4] = 0
          break
      }
      return ranks
    }

    matches = matches.map(([m, backendSelector, resource, rule]) => {
      var matchService = makeStringMatcher(m.type, m.service)
      var matchVersion = makeStringMatcher(m.type, m.version)
      var matchMethod = makeStringMatcher(m.type, m.method)
      var matchSignature = makeStringMatcher(m.type, m.signature)
      var matchFunc = function (body) {
        if (matchService && !matchService(body[1])) return false
        if (matchVersion && !matchVersion(body[2])) return false
        if (matchMethod && !matchMethod(body[3])) return false
        if (matchSignature && !matchSignature(body[4])) return false
        $matchedRoute = resource
        $matchedRule = rule
        return true
      }
      return [matchFunc, backendSelector]
    })

    return function (body) {
      var m = matches.find(([matchFunc]) => matchFunc(body))
      if (m) return m[1](body)
    }
  }

  function makeStringMatcher(type, str) {
    if (!str) return null
    switch (type) {
      case 'Exact':
        return (s) => (s === str)
      case 'RegularExpression':
        var re = new RegExp(str)
        return (s) => re.test(s)
      default: return null
    }
  }

  function makeBackendSelectorForRule(rule) {
    var selector = makeBackendSelector(
      'dubbo', listener, rule,
      function (backendRef, backendResource, filters, protocol) {
        if (!backendResource && filters.length === 0) return response500
        var forwarder = backendResource ? [makeBalancer(protocol, backendRef, backendResource)] : []
        return pipeline($=>$
          .pipe([...filters, ...forwarder], () => $ctx)
          .onEnd(() => $selection.free?.())
        )
      }
    )

    return () => selector()
  }
}
