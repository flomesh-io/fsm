(
  (
    {
      namespace,
      kind,
      name,
      pod,
    } = pipy.solve('utils.js'),
    address = os.env.REMOTE_LOGGING_ADDRESS,
    tracingLimitedID = os.env.REMOTE_LOGGING_SAMPLED_FRACTION && (os.env.REMOTE_LOGGING_SAMPLED_FRACTION * Math.pow(2, 63)),
    logLogging = address && new logging.JSONLogger('access-logger').toHTTP('http://' + address +
      (os.env.REMOTE_LOGGING_ENDPOINT || '/?query=insert%20into%20log(message)%20format%20JSONAsString'), {
      batch: {
        timeout: 1,
        interval: 1,
        prefix: '[',
        postfix: ']',
        separator: ','
      },
      headers: {
        'Content-Type': 'application/json',
        'Authorization': os.env.REMOTE_LOGGING_AUTHORIZATION || ''
      }
    }).log,
    {
      toInt63,
    } = pipy.solve('utils.js'),
    initTracingHeaders = (headers) => (
      (
        uuid = algo.uuid(),
        id = uuid.substring(0, 18).replaceAll('-', ''),
      ) => (
        headers['x-forwarded-proto'] = 'http',
        headers['x-b3-spanid'] && (
          (headers['x-b3-parentspanid'] = headers['x-b3-spanid']) && (headers['x-b3-spanid'] = id)
        ),
        !headers['x-b3-traceid'] && (
          (headers['x-b3-traceid'] = id) && (headers['x-b3-spanid'] = id)
        ),
        !headers['x-request-id'] && (
          headers['x-request-id'] = uuid
        ),
        headers['fsm-stats-namespace'] = namespace,
        headers['fsm-stats-kind'] = kind,
        headers['fsm-stats-name'] = name,
        headers['fsm-stats-pod'] = pod
      )
    )(),
  ) => (
    {
      loggingEnabled: Boolean(logLogging),

      makeLoggingData: (msg, remoteAddr, remotePort, localAddr, localPort, isOutbound) => (
        (
          sampled = false,
        ) => (
          msg?.head?.headers && (
            (isOutbound || !msg.head.headers['x-b3-traceid']) && (
              initTracingHeaders(msg.head.headers)
            ),
            sampled = (!tracingLimitedID || toInt63(msg.head.headers['x-b3-traceid']) < tracingLimitedID)
          ),
          sampled ? (
            {
              reqTime: Date.now(),
              meshName: os.env.MESH_NAME || '',
              remoteAddr,
              remotePort,
              localAddr,
              localPort,
              node: {
                ip: os.env.POD_IP || '127.0.0.1',
                name: os.env.HOSTNAME || 'localhost',
              },
              pod: {
                ns: os.env.POD_NAMESPACE || 'default',
                ip: os.env.POD_IP || '127.0.0.1',
                name: os.env.POD_NAME || os.env.HOSTNAME || 'localhost',
              },
              trace: {
                id: msg.head.headers?.['x-b3-traceid'] || '',
                span: msg.head.headers?.['x-b3-spanid'] || '',
                parent: msg.head.headers?.['x-b3-parentspanid'] || '',
                sampled: '1',
              },
              req: Object.assign({ reqSize: msg.body.size, body: msg.body.toString('base64') }, msg.head)
            }
          ) : null
        )
      )(),

      saveLoggingData: (loggingData, msg, service, target, isIngress, isEgress, type) => (
        loggingData.service = {
          name: service || 'anonymous', target: target, ingressMode: isIngress, egressMode: isEgress
        },
        loggingData.res = Object.assign({}, msg.head),
        loggingData.res['resSize'] = msg.body.size,
        loggingData.res['body'] = msg.body.toString('base64'),
        loggingData['resTime'] = Date.now(),
        loggingData['endTime'] = Date.now(),
        loggingData['type'] = type,
        logLogging(loggingData)
        // , console.log('loggingData : ', loggingData)
      ),
    }
  )
)()
