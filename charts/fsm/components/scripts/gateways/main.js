#!/usr/bin/env -S pipy --args

import options from './options.js'
import resources from './resources.js'
import { startGateway, makeResourceWatcher } from './startup.js'
import { logEnable } from './utils.js'

var opts = options(pipy.argv, {
  defaults: {
    '--config': '',
    '--watch': false,
    '--debug': false,
  },
  shorthands: {
    '-c': '--config',
    '-w': '--watch',
    '-d': '--debug',
  },
})

logEnable(opts['--debug'])
resources.init(opts['--config'], opts['--watch'] ? makeResourceWatcher() : null)
resources.list('Gateway').forEach(gw => {
  if (gw.metadata?.name) {
    startGateway(gw)
  }
})
