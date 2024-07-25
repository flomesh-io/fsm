var listenerFilterCaches = new algo.Cache(
  protocol => new algo.Cache(
    listener => makeFilters(protocol, listener?.filters)
  )
)

export default function (config, protocol, listener, rule, makeForwarder) {
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
      pipeline: makeForwarder(backendRef, backendResource, filters)
    }
  }

  function findBackendResource(backendRef) {
    if (backendRef) {
      var kind = backendRef.kind || 'Backend'
      var name = backendRef.name
      return config.resources.find(
        r => r.kind === kind && r.metadata.name === name
      )
    }
  }
}

function makeFilters(protocol, filters) {
  if (!filters) return []
  return filters.map(
    config => {
      var maker = (
        importFilter(`../config/filters/${protocol}/${config.type}.js`) ||
        importFilter(`../filters/${protocol}/${config.type}.js`)
      )
      if (!maker) throw `${protocol} filter not found: ${config.type}`
      if (typeof maker !== 'function') throw `filter ${config.type} is not a function`
      return maker(config)
    }
  )
}

function importFilter(pathname) {
  if (!pipy.load(pathname)) return null
  try {
    var filter = pipy.import(pathname)
    return filter.default
  } catch {
    return null
  }
}
