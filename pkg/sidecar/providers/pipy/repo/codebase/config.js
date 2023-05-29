(
  (
    config = JSON.decode(pipy.load('config.json')),

    uniqueTrafficMatches = {},
    uniqueTrafficMatchesFlag = {},

    uniqueIdentity = (port, protocol, destinationIPRanges,) => (
      ((id = '',) => (
        Object.keys(destinationIPRanges).sort().map(k => id += '|' + port + '_' + protocol + '_' + k + '|'), id
      ))()
    ),

    findIdentity = (id) => (
      ((key) => (
        key = Object.keys(uniqueTrafficMatches).find(k => (k.indexOf(id) >= 0)),
        key ? uniqueTrafficMatches[key] : null
      ))()
    ),

    outTrafficMatches = config?.Outbound?.TrafficMatches && Object.fromEntries(
      Object.entries(config.Outbound.TrafficMatches).map(
        ([port, match]) => [
          port,
          (
            match?.map(o => (
              (
                obj,
                stub,
              ) => (
                obj = Object.assign({}, o),
                obj.Identity = uniqueIdentity(o.Port, o.Protocol, o.DestinationIPRanges || { '*': null }),
                (stub = findIdentity(obj.Identity)) && (
                  stub.HttpHostPort2Service = Object.assign(stub.HttpHostPort2Service, obj.HttpHostPort2Service),
                  stub.HttpServiceRouteRules = Object.assign(stub.HttpServiceRouteRules, obj.HttpServiceRouteRules),
                  stub.Plugins = Object.assign(stub.Plugins, obj.Plugins),
                  obj.Identity = stub.Identity
                ),
                !stub && (uniqueTrafficMatches[obj.Identity] = obj),
                obj
              )
            )())
          )
        ]
      )
    ),

    outMergeTrafficMatches = outTrafficMatches && Object.fromEntries(
      Object.entries(outTrafficMatches).map(
        ([port, match]) => [
          port,
          match?.map(
            o => (
              uniqueTrafficMatchesFlag[o.Identity] ?
                (console.log('Merge outbound TrafficMatches : ', uniqueTrafficMatches[o.Identity]), null)
                :
                (uniqueTrafficMatchesFlag[o.Identity] = true, uniqueTrafficMatches[o.Identity])
            )
          ).filter(e => e)
        ]
      )
    ),

  ) => (

    outMergeTrafficMatches && (
      config.Outbound.TrafficMatches = outMergeTrafficMatches
    ),

    config
  )
)()