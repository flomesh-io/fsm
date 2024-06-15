export default function (config, protocol, rule, makeForwarder) {
  var ruleFilters = makeFilters(rule?.filters)

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
      ...makeFilters(backendRef?.filters),
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

  function makeFilters(filters) {
    if (!filters) return []
    return filters.map(
      config => {
        try {
          var maker = pipy.import(`../filters/${protocol}/${config.type}.js`).default
          return maker(config)
        } catch {
          throw `invalid filter type: ${config.type}`
        }
      }
    )
  }
}
