// Package flb contains controller logic for the flb
package flb

import "github.com/flomesh-io/fsm/pkg/logger"

var (
	log = logger.New("flb-utilities")
)

type TLSSecretMode string

const (
	TLSSecretModeLocal  TLSSecretMode = "local"
	TLSSecretModeRemote TLSSecretMode = "remote"
)

// FLB API paths
const (
	AuthAPIPath          = "/api/auth/local"
	UpdateServiceAPIPath = "/api/l-4-lbs/updateservice"
	DeleteServiceAPIPath = "/api/l-4-lbs/updateservice/delete"
	CertAPIPath          = "/api/updatecertificate"
	DeleteCertAPIPath    = "/api/deleteCertificates"
)
