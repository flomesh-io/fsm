package flb

import (
	"github.com/flomesh-io/fsm/pkg/logger"
)

// FLB API paths
const (
	flbAuthAPIPath          = "/api/auth/local"
	flbUpdateServiceAPIPath = "/api/l-4-lbs/updateservice"
	flbDeleteServiceAPIPath = "/api/l-4-lbs/updateservice/delete"
	flbCertAPIPath          = "/api/certificates"
	flbDeleteCertAPIPath    = "/api/deleteCertificates"
)

// FLB annotations
const (
	finalizerName        = "servicelb.flomesh.io/flb"
	flbDefaultSettingKey = "flb.flomesh.io/default-setting"
)

// FLB request HTTP headers
const (
	flbAddressPoolHeaderName    = "X-FLB-Address-Pool"
	flbDesiredIPHeaderName      = "X-FLB-Desired-Ip"
	flbMaxConnectionsHeaderName = "X-FLB-Max-Connections"
	flbReadTimeoutHeaderName    = "X-FLB-Read-Timeout"
	flbWriteTimeoutHeaderName   = "X-FLB-Write-Timeout"
	flbIdleTimeoutHeaderName    = "X-FLB-Idle-Timeout"
	flbAlgoHeaderName           = "X-FLB-Algo"
	flbUserHeaderName           = "X-FLB-User"
	flbK8sClusterHeaderName     = "X-FLB-K8s-Cluster"
	flbTagsHeaderName           = "X-FLB-Tags"
	flbTLSEnabledHeaderName     = "X-FLB-TLS-Enabled"
	flbTLSSecretHeaderName      = "X-FLB-TLS-Secret"
	flbTLSSecretModeHeaderName  = "X-FLB-TLS-Secret-Mode"
	flbTLSPortHeaderName        = "X-FLB-TLS-Port"
)

var (
	log = logger.New("flb-controller")
)

// AuthRequest is the request body for FLB authentication
type AuthRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// AuthResponse is the response body for FLB authentication
type AuthResponse struct {
	Token string `json:"jwt"`
}
