((
  { config } = pipy.solve('config.js'),

  getArgs = arg => (
    (
      args = {},
      arr,
      kv,
    ) => (
      arg && (
        (arr = arg.split('&')) && (
          arr.forEach(
            p => (
              kv = p.split('='),
              (kv.length > 1) && (
                args[kv[0].trim()] = kv[1].trim()
              )
            )
          )
        ),
        args
      )
    )
  )(),

  getCookies = cookie => (
    (
      cookies = {},
      arr,
      kv,
    ) => (
      cookie && (
        (arr = cookie.split(';')) && (
          arr.forEach(
            p => (
              kv = p.split('='),
              (kv.length > 1) && (
                cookies[kv[0].trim()] = kv[1].trim()
              )
            )
          )
        ),
        cookies
      )
    )
  )(),

  fetch_jwt_token = head => (
    (
      bearer = head?.headers?.['authorization'],
      args,
      cookies,
      jwt,
    ) => (
      bearer ? (
        (bearer?.startsWith?.('Bearer ') || bearer?.startsWith?.('bearer ')) ? (
          jwt = bearer.substring(7)
        ) : (
          jwt = bearer
        )
      ) : (
        (args = getArgs(head?.path?.split?.('?')?.[1])) && (
          jwt = args['jwt']
        ),
        !jwt && (
          (cookies = getCookies(head?.headers?.cookie)) && (
            jwt = cookies['jwt']
          )
        )
      ),
      jwt
    )
  )(),

  async_acl = (server, action, req_uri, req_token) => (
    (
      access_check_uri = '/api/v1/auth-api',
      body = { uri: req_uri, method: action, token: req_token },
    ) => (
      _promises = [
        new http.Agent(server, { connectTimeout: 1, idleTimeout: 5 })
          .request('POST', access_check_uri, { 'content-type': 'application/json; charset=utf-8', 'host': server }, JSON.encode(body))
          .then(
            msg => (
              _response = msg
            )
          )
      ]
    )
  )(),

) => pipy({
  _jwt: null,
  _addr: null,
  _json: null,
  _message: null,
  _promises: null,
  _response: null,
})

.import({
  __route: 'route',
})

.pipeline()
.branch(
  () => __route?.config?.EnableSubrequestAuthorization && (_addr = config?.Configs?.SubrequestAuthAddr), (
    $=>$
    .handleMessageStart(
      msg => (
        (_jwt = fetch_jwt_token(msg?.head)) && async_acl(_addr, msg.head.method, msg.head?.path?.split?.('?')?.[0], _jwt)
      )
    )
    .branch(
      () => _promises, (
        $=>$
        .wait(() => Promise.all(_promises))
        .handleMessageStart(
          msg => (
            (_response?.head?.status != 200) ? (
              _message = new Message({ status: 403 }, 'request to acl-server failed')
            ) : (
              _json = JSON.decode(_response?.body),
              (Object.keys(_json || {}).length > 0) ? (
                (_json.code == 200 && _json.data?.auth) ? (
                  msg.head.headers['username'] = _json.data.userName,
                  msg.head.headers['tenantcode'] = _json.data.tenantCode,
                  msg.head.headers['ai-jwt-token'] = _jwt
                ) : (
                  _message = new Message({ status: 403 }, 'status/auth error: ' + _json.code)
                )
              ) : (
                _message = new Message({ status: 403 }, 'json decode error')
              )
            )
          )
        )
        .branch(
          () => _message, (
            $=>$.replaceMessage(
              () => (
                _message
              )
            )
          ), (
            $=>$.chain()
          )
        )
      ), (
        $=>$.replaceMessage(
          () => (
            _message = new Message({ status: 403 }, 'no jwt')
          )
        )
      )
    )
  ), (
    $=>$.chain()
  )
)

)()