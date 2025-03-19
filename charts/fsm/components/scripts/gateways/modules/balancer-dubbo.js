import makeBackend from './backend.js'
import makeHealthCheck from './health-check.js'
import { log } from '../utils.js'

var $ctx
var $session
var $conn

export default function (backendRef, backendResource) {
  var backend = makeBackend(backendResource.metadata.name)
  var balancer = backend.balancer
  var hc = makeHealthCheck(backendRef, backendResource)

  var targetSelector = function () {
    $session = balancer.allocate(null, target => hc.isHealthy(target.address))
  }

  return pipeline($=>{
    $.onStart(c => { $ctx = c })
    $.pipe(evt => {
      if (evt instanceof MessageStart) {
        $ctx.backend = backend
        targetSelector(evt)
        log?.(
          `Inb #${$ctx.parent.inbound.id} Req #${$ctx.id}`, evt.head.requestID,
          `forward ${$session?.target?.address}`,
          `session ${algo.hash($session).toString(16)}`,
        )
        return $session ? forward : reject
      }
    })
    $.handleMessageStart(res => {
      var r = $ctx.response
      r.head = res.head
      r.headTime = Date.now()
    })
    $.handleMessageEnd(res => {
      var r = $ctx.response
      r.tailTime = Date.now()
    })

    if (log) {
      $.handleMessageStart(
        res => log?.(
          `Inb #${$ctx.parent.inbound.id} Req #${$ctx.id}`, $ctx.head.requestID,
          `return ${res.head.status}`,
        )
      )
    }

    var reject = pipeline($=>$
      .replaceMessage(
        new Message({ status: 500 })
      )
    )

    var forward = pipeline($=>{
      $.onStart(() => {
        $ctx.sendTime = Date.now()
        $ctx.target = $session.target.address
        $conn = {
          protocol: 'tcp',
          target: $session.target,
        }
      })
      $.mux(() => $session).to($=>$
        .encodeDubbo()
        .pipe(backend.connect, () => $conn)
        .decodeDubbo()
      )

      $.onEnd(() => $session.free())
    })
  })
}
