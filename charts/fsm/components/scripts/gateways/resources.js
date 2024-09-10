import { log, isIdentical } from './utils.js'

var DEFAULT_CONFIG_PATH = '/etc/fgw'

var resources = null
var files = {}
var secrets = {}
var updaters = {}

var notifyCreate = () => {}
var notifyDelete = () => {}
var notifyUpdate = () => {}

function init(pathname, onResourceChange) {
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

    resources = config.resources
    Object.entries(config.secrets || {}).forEach(([k, v]) => secrets[k] = v)

  } else {
    pipy.list('/config').forEach(
      pathname => {
        var pathname = os.path.join('/config', pathname)
        var data = readFile(pathname)
        if (data && data.kind && data.spec) {
          log?.(`Load resource file: ${pathname}`)
          files[pathname] = data
        }
      }
    )

    function watch() {
      pipy.watch('/config/').then(pathnames => {
        log?.('Resource files changed:', pathnames)
        pathnames.forEach(pathname => {
          changeFile(pathname, readFile(pathname))
        })
        watch()
      })
    }

    if (onResourceChange) {
      notifyCreate = function (resource) { onResourceChange(resource, null) }
      notifyDelete = function (resource) { onResourceChange(null, resource) }
      notifyUpdate = function (resource, old) { onResourceChange(resource, old) }
      watch()
    }
  }
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

function changeFile(pathname, data) {
  var old = files[pathname]
  var cur = data
  var oldKind = old?.kind
  var curKind = cur?.kind
  if (curKind && curKind === oldKind) {
    files[pathname] = cur
    notifyUpdate(cur, old)
  } else if (curKind && oldKind) {
    files[pathname] = cur
    notifyDelete(old)
    notifyCreate(cur)
  } else if (cur) {
    files[pathname] = cur
    notifyCreate(cur)
  } else if (old) {
    delete files[pathname]
    notifyDelete(old)
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

function list(kind) {
  if (resources) {
    return resources.filter(r => r.kind === kind)
  } else {
    return Object.values(files).filter(r => r.kind === kind)
  }
}

function setUpdater(kind, key, cb) {
  var listMap = (updaters[kind] ??= {})
  listMap[key] = cb ? [cb] : []
}

function addUpdater(kind, key, cb) {
  var listMap = (updaters[kind] ??= {})
  var list = (listMap[key] ??= [])
  if (!list.includes(cb)) list.push(cb)
}

function runUpdaters(kind, key, a, b, c) {
  var listMap = updaters[kind]
  if (listMap) {
    var list = listMap[key]
    if (list) {
      delete listMap[key]
      list.forEach(f => f(a, b, c))
      return true
    }
  }
  return false
}

function initZTM({ mesh, app }, onResourceChange) {
  allExports.ztm = { mesh, app }
  var resourceDir = `/users/${app.username}/resources/`
  return mesh.dir(resourceDir).then(
    paths => Promise.all(paths.map(
      pathname => readFileZTM(mesh, app, pathname).then(
        data => {
          if (data && data.kind && data.spec) {
            app.log(`Load resource file: ${pathname}`)
            files[pathname] = data
          }
        }
      )
    )).then(() => {
      function watch() {
        mesh.watch(resourceDir).then(pathnames => {
          Promise.all(pathnames.map(
            pathname => readFileZTM(mesh, app, pathname).then(
              data => {
                app.log(`Resource file changed: ${pathname}`)
                changeFile(pathname, data)
              }
            )
          ))
        }).then(() => {
          watch()
        })
      }

      if (onResourceChange) {
        notifyCreate = function (resource) { onResourceChange(resource, null) }
        notifyDelete = function (resource) { onResourceChange(null, resource) }
        notifyUpdate = function (resource, old) { onResourceChange(resource, old) }
        watch()
      }
    })
  )
}

function readFileZTM(mesh, app, pathname) {
  return mesh.read(pathname).then(
    data => {
      try {
        if (isJSON(pathname)) {
          return JSON.decode(data)
        } else if (isYAML(pathname)) {
          return YAML.decode(data)
        } else if (isSecret(pathname)) {
          var name = os.path.basename(pathname)
          secrets[name] = data
        }
      } catch {
        app.log(`Cannot load or parse file: ${pathname}, skpped.`)
      }
    }
  )
}

var allExports = {
  init,
  initZTM,
  list,
  secrets,
  setUpdater,
  addUpdater,
  runUpdaters,
  ztm: null,
}

export default allExports
