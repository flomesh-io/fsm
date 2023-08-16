// Package tresor implements the certificate.Manager interface for Tresor, a custom certificate provider in FSM.
package tresor

import (
	"math/big"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/logger"
)

const (
	// How many bits to use for the RSA key
	rsaBits = 2048

	// How many bits in the certificate serial number
	certSerialNumberBits = 128

	// The organization name to be used in the certificates
	//lint:ignore U1000 Ignore unused variable
	testCertOrgName = "Flomesh Service Mesh Tresor"
)

var (
	log               = logger.New("tresor")
	serialNumberLimit = new(big.Int).Lsh(big.NewInt(1), certSerialNumberBits)
)

// CertManager implements certificate.Manager
type CertManager struct {
	// The Certificate Authority root certificate to be used by this certificate manager
	ca                       *certificate.Certificate
	certificatesOrganization string
	keySize                  int
}
