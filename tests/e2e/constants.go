// Package e2e defines test's const vars.
package e2e

import "github.com/flomesh-io/fsm/tests/framework"

const (
	fortioImageName = "fortio/fortio"
	fortioHTTPPort  = 8080
	fortioTCPPort   = 8078
	fortioGRPCPort  = 8079

	fortioTCPRetCodeSuccess  = "OK"
	fortioGRPCRetCodeSuccess = "SERVING"
)

var (
	fortioSingleCallSpec = framework.FortioLoadTestSpec{Calls: 1}
)
