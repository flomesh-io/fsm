import resources from './resources.js'

export var log
export var dumpEnabled

var logFunc = function (a, b, c, d, e, f) {
  var n = 6
  if (f === undefined) n--
  if (e === undefined) n--
  if (d === undefined) n--
  if (c === undefined) n--
  if (b === undefined) n--
  if (a === undefined) n--
  switch (n) {
    case 0: console.log(); break
    case 1: console.log(a); break
    case 2: console.log(a, b); break
    case 3: console.log(a, b, c); break
    case 4: console.log(a, b, c, d); break
    case 5: console.log(a, b, c, d, e); break
    case 6: console.log(a, b, c, d, e, f); break
  }
}

export function enableLog(on) {
  log = on ? logFunc : null
}

export function enableDump(on) {
  dumpEnabled = on
}

export function isIdentical(a, b) {
  if (a instanceof Array && b instanceof Array) {
    return a.length === b.length && !a.some((v, i) => v !== b[i])
  } else {
    return a === b
  }
}

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

export function findPolicies(kind, targetResource) {
  return resources.list(kind).filter(
    r => r.spec.targetRefs.some(
      ref => (
        ref.kind === targetResource.kind &&
        ref.name === targetResource.metadata.name
      )
    )
  )
}

export function makeFilters(protocol, filters) {
  if (!filters) return []
  return filters.map(
    config => {
      var maker = (
        importFilter(`./config/filters/${protocol}/${config.type}.js`) ||
        importFilter(`./filters/${protocol}/${config.type}.js`)
      )
      if (!maker) throw `${protocol} filter not found: ${config.type}`
      if (typeof maker !== 'function') throw `filter ${config.type} is not a function`
      return maker(config, resources)
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
