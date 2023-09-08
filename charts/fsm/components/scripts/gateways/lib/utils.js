((
  toInt63 = str => (
    new Int("u64", new Data(str, "hex").toArray()).toString() / 2
  ),
  traceId = () => algo.uuid().substring(0, 18).replaceAll('-', ''),
) => (
  {
    namespace: (os.env.POD_NAMESPACE || 'default'),
    kind: (os.env.POD_CONTROLLER_KIND || 'Deployment'),
    name: (os.env.SERVICE_ACCOUNT || 'fgw'),
    pod: (os.env.POD_NAME || ''),

    toInt63,
    traceId,

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

  }
))()