(
  (
    config = pipy.solve('config.js'),
    {
      namespace,
      kind,
      name,
      pod,
    } = pipy.solve('utils.js'),
    tracingAddress = config?.Spec?.Observability?.tracing?.address,
    tracingEndpoint = (config?.Spec?.Observability?.tracing?.endpoint || '/api/v2/spans'),
    tracingLimitedID = config?.Spec?.Observability?.tracing?.sampledFraction && (+config.Spec.Observability.tracing.sampledFraction * Math.pow(2, 63)),
    logZipkin = tracingAddress && new logging.JSONLogger('zipkin').toHTTP('http://' + tracingAddress + tracingEndpoint, {
      batch: {
        timeout: 1,
        interval: 1,
        prefix: '[',
        postfix: ']',
        separator: ','
      },
      headers: {
        'Host': tracingAddress,
        'Content-Type': 'application/json',
      }
    }).log,
    {
      toInt63,
    } = pipy.solve('utils.js'),
  ) => (
    {
      tracingEnabled: Boolean(logZipkin),

      initTracingHeaders: (headers, proto) => (
        (
          sampled = true,
          uuid = algo.uuid(),
          id = uuid.replaceAll('-', ''),
        ) => (
          proto && (headers['x-forwarded-proto'] = proto),
          headers['x-b3-spanid'] && (
            (headers['x-b3-parentspanid'] = headers['x-b3-spanid']) && (headers['x-b3-spanid'] = id)
          ),
          !headers['x-b3-traceid'] && (
            (headers['x-b3-traceid'] = id) && (headers['x-b3-spanid'] = id)
          ),
          headers['x-b3-sampled'] && (
            sampled = (headers['x-b3-sampled'] === '1'), true
          ) || (
            (sampled = (!tracingLimitedID || toInt63(headers['x-b3-traceid']) < tracingLimitedID)) ? (headers['x-b3-sampled'] = '1') : (headers['x-b3-sampled'] = '0')
          ),
          !headers['x-request-id'] && (
            headers['x-request-id'] = uuid
          ),
          headers['fsm-stats-namespace'] = namespace,
          headers['fsm-stats-kind'] = kind,
          headers['fsm-stats-name'] = name,
          headers['fsm-stats-pod'] = pod,
          sampled
        )
      )(),

      makeZipKinData: (msg, headers, clusterName, kind, shared) => (
        (data) => (
          data = {
            'traceId': headers?.['x-b3-traceid'] && headers['x-b3-traceid'].toString(),
            'id': headers?.['x-b3-spanid'] && headers['x-b3-spanid'].toString(),
            'name': headers?.host,
            'timestamp': Date.now() * 1000,
            'localEndpoint': {
              'port': 0,
              'ipv4': os.env.POD_IP || '',
              'serviceName': name,
            },
            'tags': {
              'component': 'proxy',
              'http.url': headers?.['x-forwarded-proto'] + '://' + headers?.host + msg?.head?.path,
              'http.method': msg?.head?.method,
              'node_id': os.env.POD_UID || '',
              'http.protocol': msg?.head?.protocol,
              'guid:x-request-id': headers?.['x-request-id'],
              'user_agent': headers?.['user-agent'],
              'upstream_cluster': clusterName
            },
            'annotations': []
          },
          headers['x-b3-parentspanid'] && (data['parentId'] = headers['x-b3-parentspanid']),
          data['kind'] = kind,
          shared && (data['shared'] = shared),
          data.tags['request_size'] = '0',
          data.tags['response_size'] = '0',
          data.tags['http.status_code'] = '502',
          data.tags['peer.address'] = '',
          data['duration'] = 0,
          data
        )
      )(),

      saveTracing: (zipkinData, messageHead, bytesStruct, target) => (
        zipkinData && (
          zipkinData.tags['peer.address'] = target,
          zipkinData.tags['http.status_code'] = messageHead.status?.toString?.(),
          zipkinData.tags['request_size'] = bytesStruct.requestSize.toString(),
          zipkinData.tags['response_size'] = bytesStruct.responseSize.toString(),
          zipkinData['duration'] = Date.now() * 1000 - zipkinData['timestamp'],
          logZipkin(zipkinData)
          // , console.log('zipkinData : ', zipkinData)
        )
      ),
    }
  )
)()
