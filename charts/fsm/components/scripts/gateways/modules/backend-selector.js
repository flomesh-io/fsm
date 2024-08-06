import resources from '../resources.js'
import { makeFilters } from '../utils.js'

var listenerFilterCaches = new algo.Cache(
  protocol => new algo.Cache(
    listener => makeFilters(protocol, listener?.routeFilters)
  )
)

export default function (protocol, listener, rule, makeBalancer) {
  var ruleFilters = [
    ...listenerFilterCaches.get(protocol).get(listener),
    ...makeFilters(protocol, rule?.filters),
  ]

  var refs = rule?.backendRefs || []
  if (refs.length > 1) {
    var lb = new algo.LoadBalancer(
      refs.map(ref => makeBackendTarget(ruleFilters, ref)),
      {
        key: t => t.id,
        weight: t => t.weight,
      }
    )
    return (hint) => lb.allocate(hint)
  } else {
    var singleSelection = { target: makeBackendTarget(ruleFilters, refs[0]) }
    return () => singleSelection
  }

  function makeBackendTarget(ruleFilters, backendRef) {
    var backendResource = findBackendResource(backendRef)
    var filters = [
      ...ruleFilters,
      ...makeFilters(protocol, backendRef?.filters),
    ]
    return {
      id: backendRef?.name,
      backendRef,
      backendResource,
      weight: backendRef?.weight || 1,
      pipeline: makeBalancer(backendRef, backendResource, filters)
    }
  }

  function findBackendResource(backendRef) {
    if (backendRef) {
      var kind = backendRef.kind || 'Backend'
      var name = backendRef.name
      return resources.list(kind).find(
        r => r.metadata.name === name
      )
    }
  }
}
