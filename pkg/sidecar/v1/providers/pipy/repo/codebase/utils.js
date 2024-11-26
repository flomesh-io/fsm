((
  namespace = (os.env.POD_NAMESPACE || 'default'),
  kind = (os.env.POD_CONTROLLER_KIND || 'Deployment'),
  name = (os.env.SERVICE_ACCOUNT || ''),
  pod = (os.env.POD_NAME || ''),
  hexChar = { '0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9, 'a': 10, 'b': 11, 'c': 12, 'd': 13, 'e': 14, 'f': 15 },
  toInt63 = str => (
    (
      value = str.split('').reduce((calc, char) => (calc * 16) + hexChar[char], 0),
    ) => value / 2
  )(),
  traceId = () => algo.uuid().substring(0, 18).replaceAll('-', ''),
) => (
  {
    namespace,
    kind,
    name,
    pod,

    initRateLimit: rateLimit => (
      rateLimit?.Local ? (
        {
          count: 0,
          backlog: rateLimit.Local.Backlog || 0,
          quota: new algo.Quota(
            rateLimit.Local.Burst || rateLimit.Local.Requests || 0,
            {
              produce: rateLimit.Local.Requests || 0,
              per: rateLimit.Local.StatTimeWindow || 0,
            }
          ),
          response: new Message({
            status: rateLimit.Local.ResponseStatusCode || 429,
            headers: Object.fromEntries((rateLimit.Local.ResponseHeadersToAdd || []).map(({ Name, Value }) => [Name, Value])),
          }),
        }
      ) : null
    ),

    shuffle: arg => (
      (
        sort = a => (a.map(e => e).map(() => a.splice(Math.random() * a.length | 0, 1)[0])),
      ) => (
        arg ? Object.fromEntries(sort(sort(Object.entries(arg)))) : {}
      )
    )(),

    failover: json => (
      json ? ((obj = null) => (
        obj = Object.fromEntries(
          Object.entries(json).map(
            ([k, v]) => (
              (v === 0) ? ([k, 1]) : null
            )
          ).filter(e => e)
        ),
        Object.keys(obj).length === 0 ? null : new algo.RoundRobinLoadBalancer(obj)
      ))() : null
    ),

    toInt63,
    traceId,
  }
))()
