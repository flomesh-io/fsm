(
  (
    { config } = pipy.solve('config.js'),

    ruLogging = config?.Configs?.ResourceUsage?.StorageAddress && new logging.JSONLogger('resource-usage-logger').toHTTP(config.Configs.ResourceUsage.StorageAddress, {
      batch: {
        timeout: 1,
        interval: 1,
        prefix: '[',
        postfix: ']',
        separator: ','
      },
      headers: {
        'Content-Type': 'application/json',
        'Authorization': config.Configs.ResourceUsage.Authorization || ''
      }
    }).log,

    k8s_cluster = os.env.PIPY_K8S_CLUSTER || '',
    code_base = pipy.source || '',
    pipy_id = pipy.name || '',

    { metrics } = pipy.solve('lib/metrics.js'),
    cpuUsage = (
      (
        items = os.readFile('/proc/self/stat')?.toString?.()?.split?.(" "),
        su = os.readFile('/proc/uptime')?.toString?.()?.split?.(".")?.[0],
        dr,
        ur,
      ) => (
        items && su && (
          dr = su - items[21] / 100,
          ur = +items[13] + +items[14],
          (ur / (dr < 0 ? 1 : dr)).toFixed(2)
        )
      )
    ),
    memSize = os.readFile('/proc/meminfo')?.toString?.()?.split?.('\n')?.filter?.(s => s.startsWith('MemTotal'))?.[0]?.split?.(' ')?.filter?.(e => e)?.[1],
    memUsage = (
      (
        ram = os.readFile('/proc/self/statm')?.toString?.()?.split?.(' ')?.[1],
      ) => (
        (+ram * 4 * 100 / memSize).toFixed(2)
      )
    ),
    hostname = pipy.exec('hostname')?.toString?.()?.replaceAll?.('\n', ''),
    cpuUsageMetric = metrics.fgwResourceUsage.withLabels(pipy.uuid || '', pipy.name || '', pipy.source || '', hostname, 'cpu'),
    memUsageMetric = metrics.fgwResourceUsage.withLabels(pipy.uuid || '', pipy.name || '', pipy.source || '', hostname, 'mem'),
  ) => pipy({
    _cpu: null,
    _mem: null,
  })

.pipeline()
.task(config.Configs.ResourceUsage.ScrapeInterval + 's')
.onStart(
  () => (
    cpuUsageMetric.set(_cpu = +cpuUsage()),
    memUsageMetric.set(_mem = +memUsage()),
    ruLogging?.(
      {
        k8s_cluster,
        code_base,
        pipy_id,
        host: hostname,
        cpu: _cpu,
        mem: _mem,
      }
    ),
    new StreamEnd
  )
)

)()