((
  config = pipy.solve('config.js'),
  dnsServers = { primary: config?.Spec?.LocalDNSProxy?.UpstreamDNSServers?.Primary, secondary: config?.Spec?.LocalDNSProxy?.UpstreamDNSServers?.Secondary },
  dnsSvcAddress = (os.env.PIPY_NAMESERVER || dnsServers?.primary || dnsServers?.secondary) && ((os.env.PIPY_NAMESERVER || dnsServers?.primary || dnsServers?.secondary) + ":53"),
  dnsRecordSets = {},
  dnsIPv6RecordSets = {},
  dnsCache = new algo.Cache(null, null, { ttl: 10 }),
) => (
  config?.DNSResolveDB && (
    Object.entries(config.DNSResolveDB).map(
      ([k, v]) => (
        (rr, rrv6) => (
          rr = [],
          rrv6 = [],
          v.map(
            ip => (
              new IP(ip).version === 4 ? rr.push({
                'name': k,
                'type': 'A',
                'ttl': 600, // TTL : 10 minutes
                'rdata': ip
              }) :
              rrv6.push({
                'name': k,
                'type': 'AAAA',
                'ttl': 600, // TTL : 10 minutes
                'rdata': ip
              })
            )
          ),
          dnsRecordSets[k] = rr,
          dnsIPv6RecordSets[k] = rrv6
        )
      )()
    )
  ),

  pipy({
    _response: null,
    _alternative: null,
    _remoteAddressPort: null,
  })

  .pipeline()
  .handleStreamStart(
    () => _remoteAddressPort = __inbound.remoteAddress + ':' + __inbound.remotePort
  )
  .replaceData(
    dat => (
      _response = null,
      ((dns, answer, asterisk, record) => (
        dns = DNS.decode(dat),
        (dns?.question?.[0]?.type === 'A') ? (
          (answer = dnsRecordSets[dns?.question?.[0]?.name] || (asterisk = true) && dnsRecordSets['*'])
        ) : (dns?.question?.[0]?.type === 'AAAA') && (
          (answer = dnsIPv6RecordSets[dns?.question?.[0]?.name] || (asterisk = true) && dnsIPv6RecordSets['*'])
        ),
        answer ? (
          dns.qr = 1,
          dns.rd = 1,
          dns.ra = 1,
          dns.aa = 1,
          dns.rcode = 0,
          dns.question = [{
            'name': dns.question[0].name,
            'type': dns.question[0].type
          }],
          asterisk ? (
            dns.answer = answer.map(a => (record = { ...a }, record.name = dns.question[0].name, record))
          ) : (
            dns.answer = answer
          ),
          dns.authority = [],
          dns.additional = [],
          _response = new Data(DNS.encode(dns)),
          dnsCache.set(_remoteAddressPort + '@' + dns.id, _response)
        ) : !dnsSvcAddress && (
          dns.qr = 1,
          dns.rd = 1,
          dns.ra = 1,
          dns.rcode = 3,
          dns.question = [{
            'name': dns.question[0].name,
            'type': dns.question[0].type
          }],
          dns.answer = [],
          dns.authority = [],
          dns.additional = [],
          _response = new Data(DNS.encode(dns))
        )
      ))(),
      // _response ? _response : dat
      // _response && (_alternative = _response, _response = null), dat
      !dnsSvcAddress ? (
        _response
      ) : (
        _response = null, dat
      )
    )
  )
  .branch(
    () => !Boolean(_response), $ => $
      .connect(() => dnsSvcAddress, { protocol: 'udp' })
      .replaceData(
        dat => ((dns, name, type, nsname, fake = false) => (
          dns = DNS.decode(dat),
          (dns?.rcode === 3 || (!Boolean(dns?.answer) && !Boolean(dns?.authority))) && (
            name = dns?.question?.[0]?.name,
            type = dns?.question?.[0]?.type,
            name && type && (
              fake = true
            ),
            (dns?.authority?.length > 0 && (nsname = dns?.authority?.[0]?.name)) && (
              // exclude domain suffix : search svc.cluster.local cluster.local
              name.endsWith('.cluster.local') && nsname && (fake = false)
            ),
            fake = false, // disable fake response
            (_alternative = dnsCache.get(_remoteAddressPort + '@' + dns.id)) && (fake = false)
          ),

          fake && (
            dns.qr = 1,
            dns.rd = 1,
            dns.ra = 1,
            dns.aa = 1,
            dns.rcode = 0,
            dns.question = [{
              'name': name,
              'type': type
            }],
            dns.authority = [{
              'name': name,
              'type': 'SOA',
              'ttl': 1800,
              'rdata': {
                'mname': 'localhost',
                'rname': 'admin.localhost',
                'serial': 1663232447,
                'refresh': 1800,
                'retry': 900,
                'expire': 604800,
                'minimum': 86400
              }
            }],
            dns.additional = [
              {
                'name': '',
                'type': 'OPT',
                'class': 1232,
                'ttl': 0,
                'rdata': ''
              }
            ],
            // ipv4 : 127.0.0.2
            (type === 'A') && (
              dns.answer = [{
                'name': name,
                'type': type,
                'ttl': 5400,
                'rdata': '127.0.0.2'
              }]
            ),
            // ipv6 : ::ffff:127.0.0.2
            (type == 'AAAA') && (
              dns.answer = [{
                'name': name,
                'type': type,
                'ttl': 5400,
                'rdata': '00000000000000000000ffff7f000002'
              }]
            )
          ),
          _alternative ? _alternative : (fake ? new Data(DNS.encode(dns)) : dat)
        ))()
      ),
    $ => $
  )

))()