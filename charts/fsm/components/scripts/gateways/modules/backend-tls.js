import resources from '../resources.js'
import { findPolicies } from '../utils.js'

export default function (backendRef, backendResource, gateway) {
  var backendTLSPolicies = findPolicies('BackendTLSPolicy', backendResource)
  var tlsCertificate = gateway.spec.backendTLS?.clientCertificate
  var tlsValidationConfig = backendTLSPolicies.find(r => r.spec.validation)?.spec?.validation
  var tlsConfig = null

  if (tlsCertificate) {
    tlsConfig = tlsConfig || {}
    tlsConfig.certificate = {
      cert: new crypto.Certificate(resources.secrets[tlsCertificate['tls.crt']]),
      key: new crypto.PrivateKey(resources.secrets[tlsCertificate['tls.key']]),
    }
  }

  if (tlsValidationConfig) {
    tlsConfig = tlsConfig || {}
    tlsConfig.sni = tlsValidationConfig.hostname
    tlsConfig.trusted = tlsValidationConfig.caCertificates?.map?.(
      c => new crypto.Certificate(
        resources.secrets[c['ca.crt']]
      )
    )
    if (tlsValidationConfig.subjectAltNames) {
      var names = tlsValidationConfig.subjectAltNames.map(i => i.type === 'URI' ? i.uri : i.hostname)
      tlsConfig.onVerify = function (ok, cert) {
        if (!ok) return false
        return cert.subjectAltNames.some(name => names.includes(name))
      }
    } else {
      var hostname = tlsValidationConfig.hostname
      tlsConfig.onVerify = function (ok, cert) {
        if (!ok) return false
        return cert.subject?.commonName === hostname
      }
    }
  }
  return tlsConfig
}
