var DEFAULT_CONFIG_PATH = '/etc/fgw'

function load(filename) {
  if (!filename) {
    if (os.stat(DEFAULT_CONFIG_PATH)?.isDirectory?.()) {
      filename = DEFAULT_CONFIG_PATH
    } else {
      loadConfig(JSON.decode(pipy.load('config.json')))
      return
    }
  }

  var st = os.stat(filename)
  if (!st) throw `file or directory not found: ${filename}`

  if (st.isDirectory()) {
    loadConfigDir(filename)
  } else {
    if (isYAML(filename)) {
      loadConfig(YAML.decode(os.read(filename)))
    } else {
      loadConfig(JSON.decode(os.read(filename)))
    }
  }
}

function loadConfig(obj) {
  config.resources = obj.resources || []
  config.secrets = obj.secrets || {}
}

function loadConfigDir(dirname) {
  var list = os.readDir(dirname)
  var resources = []
  var secrets = {}
  list.forEach(name => {
    var filename = os.path.join(dirname, name)
    if (isSecret(name)) {
      secrets[name] = os.read(filename).toString()
    } else if (isJSON(name)) {
      resources.push(JSON.decode(os.read(filename)))
    } else if (isYAML(name)) {
      resources.push(YAML.decode(os.read(filename)))
    }
  })
  config.resources = resources
  config.secrets = secrets
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

var config = {
  load,
  resources: null,
  secrets: null,
}

export default config
