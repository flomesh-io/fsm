import { log } from '../log.js'

export default function (config, protocol, rule, makeForwarder) {
  var ruleFilters = makeFilters(rule?.filters)

  var refs = rule?.backendRefs || []
  log?.('rule', rule, 'refs', refs)

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
    log?.('[makeBackendTarget]','backendRef', backendRef, 'backendResource', backendResource)

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
      var res = config.resources.find(
        r => r.kind === kind && r.metadata.name === name
      )
      log?.('[findBackendResource]', 'backendRef', backendRef, 'backendResource', res)

      return res
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
