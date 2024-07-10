import { findPolicies } from '../utils.js'

export default function (config, backendRef, backendResource) {
  var backendTLSPolicies = findPolicies(config, 'BackendTLSPolicy', backendResource)
  var tlsValidationConfig = backendTLSPolicies.find(r => r.spec.validation)?.spec?.validation
  return tlsValidationConfig && {
    sni: tlsValidationConfig.hostname,
    trusted: tlsValidationConfig.caCertificates.map(
      c => new crypto.Certificate(
        config.secrets[c['ca.crt']]
      )
    )
  }
}
