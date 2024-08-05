import { log } from './utils.js'

var DEFAULT_CONFIG_PATH = '/etc/fgw'

var files = {}
var kinds = {}
var secrets = {}

var notifyCreate = () => {}
var notifyDelete = () => {}
var notifyUpdate = () => {}

function init(pathname, onChange) {
  var configFile = pipy.load('/config.json') || pipy.load('/config.yaml')
  var configDir = pipy.list('/config')
  var hasBuiltinConfig = (configDir.length > 0 || Boolean(configFile))

  if (pathname || !hasBuiltinConfig) {
    pathname = os.path.resolve(pathname || DEFAULT_CONFIG_PATH)
    var s = os.stat(pathname)
    if (!s) {
      throw `configuration file or directory does not exist: ${pathname}`
    }
    if (s.isDirectory()) {
      pipy.mount('config', pathname)
      configFile = null
    } else if (s.isFile()) {
      configFile = os.read(pathname)
    }
  }

  if (configFile) {
    var config
    try {
      try {
        config = JSON.decode(configFile)
      } catch {
        config = YAML.decode(configFile)
      }
    } catch {
      throw 'cannot parse configuration file as JSON or YAML'
    }

    config.resources.forEach(r => {
      if (r.kind) {
        list(r.kind).push(r)
      }
    })

    Object.entries(config.secrets || {}).forEach(([k, v]) => secrets[k] = v)

  } else {
    notifyCreate = function (resource) { onChange(resource, null) }
    notifyDelete = function (resource) { onChange(null, resource) }
    notifyUpdate = function (resource, old) { onChange(resource, old) }

    pipy.list('/config').forEach(
      pathname => addFile(pathname)
    )
  }
}

function list(kind) {
  return (kinds[kind] ??= [])
}

function readFile(pathname) {
  try {
    if (isJSON(pathname)) {
      return JSON.decode(pipy.load(pathname))
    } else if (isYAML(pathname)) {
      return YAML.decode(pipy.load(pathname))
    } else if (isSecret(pathname)) {
      var name = os.path.basename(pathname)
      secrets[name] = pipy.load(pathname)
    }
  } catch {
    console.error(`Cannot load or parse file: ${pathname}, skpped.`)
  }
}

function addFile(pathname) {
  var data = readFile(pathname)
  if (data && data.kind && data.spec) {
    log?.(`Load resource file: ${pathname}`)
    files[pathname] = data
    var resources = list(data.kind)
    var name = data.metadata?.name
    if (name) {
      var i = resources.find(r => r.metadata?.name === name)
      if (i >= 0) {
        var old = resources[i]
        resources[i] = data
        notifyUpdate(data, old)
      } else {
        resources.push(data)
        notifyCreate(data)
      }
    } else {
      resources.push(data)
      notifyCreate(data)
    }
  }
}

function isJSON(filename) {
  return filename.endsWith('.json')
}

function isYAML(filename) {
  return filename.endsWith('.yaml') || filename.endsWith('.yml')
}

function isSecret(filename) {
  return filename.endsWith('.crt') || filename.endsWith('.key')
}

var updaterLists = []

function findUpdaterKey(key) {
  if (key instanceof Array) {
    updaterLists.findIndex(([k]) => (
      k.length === key.length &&
      !k.some((v, i) => (v !== key[i]))
    ))
  } else {
    updaterLists.findIndex(([k]) => (k === key))
  }
}

function addUpdater(key, updater) {
  var i = findUpdaterKey(key)
  if (i >= 0) {
    updaterLists[i][1].push(updater)
  } else {
    if (key instanceof Array) key = [...key]
    updaterLists.push([key, [updater]])
  }
}

function getUpdaters(key) {
  var i = findUpdaterKey(key)
  if (i >= 0) return updaterLists[i][1]
  return []
}

export default {
  init,
  list,
  secrets,
  addUpdater,
  getUpdaters,
}
