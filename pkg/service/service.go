package service

var (
	trustDomain = `cluster.local`
)

// SetTrustDomain sets the trust domain
func SetTrustDomain(domain string) {
	if len(domain) > 0 {
		trustDomain = domain
	}
}

// GetTrustDomain returns the trust domain
func GetTrustDomain() string {
	return trustDomain
}

// ServerName returns the Server Name Identifier (SNI) for TLS connections
func (ms MeshService) ServerName() string {
	return ms.FQDN()
}
