((
  { config, isDebugEnabled } = pipy.solve('config.js'),

  { getNonNegativeNumber } = pipy.solve('lib/utils.js'),

  clientMaxBodySizeCache = new algo.Cache(
    route => (
      typeof (route?.config?.ClientMaxBodySize) !== 'undefined' ? (
        getNonNegativeNumber(route.config.ClientMaxBodySize)
      ) : (
        typeof (__domain?.ClientMaxBodySize) !== 'undefined' ? (
          getNonNegativeNumber(__domain.ClientMaxBodySize)
        ) : (
          typeof (config?.Configs?.ClientMaxBodySize) !== 'undefined' ? (
            getNonNegativeNumber(config.Configs.ClientMaxBodySize)
          ) : 1000000 // 1m
        )
      )
    )
  ),

  makeMatchDomainHandler = portRouteRules => (
    (
      domains = {},
      starDomains = [],
      lowercaseDomain,
    ) => (
      Object.keys(portRouteRules || {}).forEach(
        domain => (
          portRouteRules[domain].name = domain,
          lowercaseDomain = domain.toLowerCase(),
          domain.startsWith('*') ? (
            portRouteRules[domain].starName = lowercaseDomain.substring(1),
            starDomains.push(portRouteRules[domain])
          ) : (
            domains[lowercaseDomain] = portRouteRules[domain]
          )
        )
      ),
      domain => (
        domains[domain] ? (
          domains[domain]
        ) : (
          starDomains.find(
            d => (
              domain.endsWith(d.starName) ? d : null
            )
          )
        )
      )
    )
  )(),

  matchDomainHandlers = new algo.Cache(makeMatchDomainHandler),

  getParameters = path => (
    (
      params = {},
      qsa,
      qs,
      arr,
      kv,
    ) => (
      path && (
        (qsa = path.split('?')[1]) && (
          (qs = qsa.split('#')[0]) && (
            (arr = qs.split('&')) && (
              arr.forEach(
                p => (
                  kv = p.split('='),
                  params[kv[0]] = kv[1]
                )
              )
            )
          )
        )
      ),
      params
    )
  )(),

  makeDictionaryMatches = dictionary => (
    (
      tests = Object.entries(dictionary || {}).map(
        ([type, dict]) => (
          (type === 'Exact') ? (
            Object.keys(dict || {}).map(
              k => (obj => obj?.[k] === dict[k])
            )
          ) : (
            (type === 'Regex') ? (
              Object.keys(dict || {}).map(
                k => (
                  (
                    regex = new RegExp(dict[k])
                  ) => (
                    obj => regex.test(obj?.[k] || '')
                  )
                )()
              )
            ) : [() => false]
          )
        )
      )
    ) => (
      (tests.length > 0) && (
        obj => tests.every(a => a.every(f => f(obj)))
      )
    )
  )(),

  pathPrefix = (path, prefix) => (
    path.startsWith(prefix) && (
      prefix.endsWith('/') || (
        (
          lastChar = path.charAt(prefix.length),
        ) => (
          lastChar === '' || lastChar === '/'
        )
      )()
    )
  ),

  makeHttpMatches = rule => (
    (
      matchPath = (
        (rule?.Path?.Type === 'Regex') && (
          ((match = null) => (
            match = new RegExp(rule?.Path?.Path),
            (path) => match.test(path)
          ))()
        ) || (rule?.Path?.Type === 'Exact') && (
          (path) => path === rule?.Path?.Path
        ) || (rule?.Path?.Type === 'Prefix') && (
          (path) => pathPrefix(path, rule?.Path?.Path)
        ) || rule?.Path?.Type && (
          () => false
        )
      ),
      matchHeaders = makeDictionaryMatches(rule?.Headers),
      matchMethod = (
        rule?.Methods && Object.fromEntries((rule.Methods).map(m => [m, true]))
      ),
      matchParams = makeDictionaryMatches(rule?.QueryParams),
    ) => (
      {
        config: rule,
        match: message => (
          (!matchMethod || matchMethod[message?.head?.method]) && (
            (!matchPath || matchPath(message?.head?.path?.split('?')[0])) && (
              (!matchHeaders || matchHeaders(message?.head?.headers)) && (
                (!matchParams || matchParams(getParameters(message?.head?.path)))
              )
            )
          )
        ),
        backendServiceBalancer: new algo.RoundRobinLoadBalancer(Object.fromEntries(Object.entries(rule?.BackendService || {})
          .map(([k, v]) => [k, v.Weight])
          .filter(([k, v]) => v > 0)
        )),
        ...(rule?.ServerRoot && { serverRoot: rule.ServerRoot })
      }
    )
  )(),

  makeGrpcMatches = rule => (
    (
      matchHeaders = makeDictionaryMatches(rule?.Headers),
      matchMethod = (
        rule?.Method?.Type === 'Exact' && (
          path => (
            (
              grpc = (path || '').split('/'),
            ) => (
              path?.startsWith('/grpc.reflection.') || (
                (!rule?.Method?.Service || rule?.Method?.Service === grpc[1]) && (!rule?.Method?.Method || rule?.Method?.Method === grpc[2])
              )
            )
          )()
        )
      ),
    ) => (
      {
        config: rule,
        match: message => (
          (!matchHeaders || matchHeaders(message?.head?.headers)) && (
            (!matchMethod || matchMethod(message?.head?.path))
          )
        ),
        backendServiceBalancer: new algo.RoundRobinLoadBalancer(Object.fromEntries(Object.entries(rule?.BackendService || {})
          .map(([k, v]) => [k, v.Weight])
          .filter(([k, v]) => v > 0)
        )),
      }
    )
  )(),

  makeRouteMatchesHandler = routeTypeMatches => (
    (
      matches = [],
    ) => (
      (!routeTypeMatches?.RouteType || routeTypeMatches?.RouteType === 'HTTP' || routeTypeMatches?.RouteType === 'HTTP2') && (
        matches = (routeTypeMatches?.Matches || []).map(
          m => makeHttpMatches(m)
        )
      ),
      (routeTypeMatches?.RouteType === 'GRPC') && (
        matches = (routeTypeMatches?.Matches || []).map(
          m => makeGrpcMatches(m)
        )
      ),
      message => (
        matches.find(
          m => m.match(message)
        )
      )
    )
  )(),

  routeMatchesHandlers = new algo.Cache(makeRouteMatchesHandler),

  portCache = new algo.Cache(
    port => config?.RouteRules?.[port] && (
      new algo.Cache(
        host => (
          (
            routeRules = config.RouteRules[port],
            matchDomain = matchDomainHandlers.get(routeRules),
            domain = matchDomain(host.toLowerCase()),
            messageHandler = routeMatchesHandlers.get(domain),
          ) => (
            message => (
              _host = host,
              __domain = domain,
              messageHandler && (
                __route = messageHandler(message)
              )
            )
          )
        )()
      )
    )
  ),

  handleMessage = (host, msg) => (
    host && (
      (
        hostHandlers = portCache.get(__port?.Port),
        handler = hostHandlers && hostHandlers.get(host),
      ) => (
        handler && handler(msg)
      )
    )()
  ),

) => pipy({
  _host: undefined,
  _continue: true,
  _clientBodySize: 0,
  _clientMaxBodySize: 0,
})

.export('route', {
  __domain: null,
  __route: null,
})

.import({
  __port: 'listener',
  __consumer: 'consumer',
})

.pipeline()
.handleMessageStart(
  msg => (
    handleMessage(msg?.head?.headers?.host, msg),
    !__domain && __consumer?.sni && (
      handleMessage(__consumer.sni, msg)
    ),
    !__domain && config?.Configs?.StripAnyHostPort && (
      handleMessage(msg?.head?.headers?.host?.split(':')?.[0], msg)
    ),
    __route && (__domain.RouteType === 'HTTP') && (
      _clientMaxBodySize = clientMaxBodySizeCache.get(__route)
    )
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('[route] port, host, __domain.name, __route.path:', __port?.Port, _host, __domain?.name, __route?.config?.Path?.Path || __route?.config?.Method)
      )
    )
  )
)
.branch(
  () => (_clientMaxBodySize > 0), (
    $=>$
    .replaceData(
      dat => (
        _clientBodySize += dat.size,
        _clientBodySize > _clientMaxBodySize ? (
          _continue = false,
          null
        ) : dat
      )
    )
    .replaceMessage(
      msg => (
        !_continue ? (
          new Message({ status: 413 }, 'Request Entity Too Large')
        ) : msg
      )
    )
  ), (
    $=>$
  )
)
.branch(
  () => _continue, (
    $=>$.chain()
  ), (
    $=>$
  )
)

)()