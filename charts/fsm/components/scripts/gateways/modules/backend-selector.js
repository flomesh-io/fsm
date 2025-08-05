import resources from '../resources.js'
import { makeFilters } from '../utils.js'

var listenerFilterCaches = new algo.Cache(
  protocol => new algo.Cache(
    listener => makeFilters(protocol, listener?.routeFilters)
  )
)

export default function (protocol, listener, rule, makeBalancer) {
  var routeFilters = listenerFilterCaches.get(protocol).get(listener)
  var ruleFilters = makeFilters(routeFilters.outputProtocol || protocol, rule?.filters)

  var refs = rule?.backendRefs || []
  if (refs.length > 1) {
    var lb = new algo.LoadBalancer(
      refs.map(ref => makeBackendTarget(ref)),
      {
        key: t => t.id,
        weight: t => t.weight,
      }
    )
    return (hint) => lb.allocate(hint)
  } else {
    var singleSelection = { target: makeBackendTarget(refs[0]) }
    return () => singleSelection
  }

  function makeBackendTarget(backendRef) {
    var backendResource = findBackendResource(backendRef)
    var backendFilters = makeFilters(ruleFilters.outputProtocol || protocol, backendRef?.filters)
    var filters = [
      ...routeFilters,
      ...ruleFilters,
      ...backendFilters,
    ]
    return {
      id: backendRef?.name,
      backendRef,
      backendResource,
      weight: backendRef?.weight || 1,
      pipeline: makeBalancer(backendRef, backendResource, filters, backendFilters.outputProtocol || protocol)
    }
  }

  function findBackendResource(backendRef) {
    if (backendRef) {
      var kind = backendRef.kind || 'Backend'
      var name = backendRef.name
      return resources.find(kind, name)
    }
  }
}
