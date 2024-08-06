import resources from '../resources.js'
import { findPolicies } from '../utils.js'

export default function (backendRef, backendResource) {
  var backendTLSPolicies = findPolicies('BackendTLSPolicy', backendResource)
  var tlsValidationConfig = backendTLSPolicies.find(r => r.spec.validation)?.spec?.validation
  return tlsValidationConfig && {
    sni: tlsValidationConfig.hostname,
    trusted: tlsValidationConfig.caCertificates.map(
      c => new crypto.Certificate(
        resources.secrets[c['ca.crt']]
      )
    )
  }
}
