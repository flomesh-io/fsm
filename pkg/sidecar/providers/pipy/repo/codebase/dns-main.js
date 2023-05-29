((
  config = pipy.solve('config.js'),
  dnsServers = { primary: config?.Spec?.LocalDNSProxy?.UpstreamDNSServers?.Primary, secondary: config?.Spec?.LocalDNSProxy?.UpstreamDNSServers?.Secondary },
  dnsSvcAddress = (dnsServers?.primary || dnsServers?.secondary || os.env.LOCAL_DNS_PROXY_PRIMARY_UPSTREAM || '10.96.0.10') + ":53",
  dnsRecordSets = {},
) => (
  config?.DNSResolveDB && (
    Object.entries(config.DNSResolveDB).map(
      ([k, v]) => (
        ((rr) => (
          rr = [],
          v.map(
            ip => (
              rr.push({
                'name': k,
                'type': 'A',
                'ttl': 600, // TTL : 10 minutes
                'rdata': ip
              })
            )
          ),
          dnsRecordSets[k] = rr
        ))()
      )
    )
  ),

  pipy({
    _response: null
  })

  .pipeline()
  .replaceMessage(
    msg => (
      _response = null,
      ((dns, answer) => (
        dns = DNS.decode(msg.body),
        (dns?.question?.[0]?.type === 'A') && (
          (answer = dnsRecordSets[dns?.question?.[0]?.name]) && (
            dns.qr = 1,
            dns.rd = 1,
            dns.ra = 1,
            dns.aa = 1,
            dns.rcode = 0,
            dns.question = [{
              'name': dns.question[0].name,
              'type': dns.question[0].type
            }],
            dns.answer = answer,
            dns.authority = [],
            dns.additional = [],
            _response = new Message(DNS.encode(dns))
          )
        )
      ))(),
      _response ? _response : msg
    )
  )
  .branch(
    () => !Boolean(_response), $ => $
      .connect(() => dnsSvcAddress, { protocol: 'udp' })
      .replaceMessage(
        msg => ((dns, name, type, nsname, fake = false) => (
          dns = DNS.decode(msg.body),

          (dns?.rcode === 3 || (!Boolean(dns?.answer) && !Boolean(dns?.authority))) && (
            name = dns?.question?.[0]?.name,
            type = dns?.question?.[0]?.type,
            name && type && (
              fake = true
            ),
            (dns?.authority?.length > 0 && (nsname = dns?.authority?.[0]?.name)) && (
              // exclude domain suffix : search svc.cluster.local cluster.local
              name.endsWith('.cluster.local') && nsname && (fake = false)
            )
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

          fake ? new Message(DNS.encode(dns)) : msg
        ))()
      ),
    $ => $
  )

))()