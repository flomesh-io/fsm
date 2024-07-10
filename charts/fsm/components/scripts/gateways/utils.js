export function stringifyHTTPHeaders(headers) {
  return Object.entries(headers).flatMap(
    ([k, v]) => {
      if (v instanceof Array) {
        return v.map(v => `${k}=${v}`)
      } else {
        return `${k}=${v}`
      }
    }
  ).join(' ')
}

export function findPolicies(config, kind, targetResource) {
  return config.resources.filter(
    r => {
      if (r.kind !== kind) return false
      return r.spec.targetRefs.some(
        ref => (
          ref.kind === targetResource.kind &&
          ref.name === targetResource.metadata.name
        )
      )
    }
  )
}
