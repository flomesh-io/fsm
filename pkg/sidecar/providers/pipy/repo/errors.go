package repo

import (
	"fmt"
)

var errTooManyConnections = fmt.Errorf("too many connections")
var errServiceAccountMismatch = fmt.Errorf("service account mismatch in nodeid vs xds certificate common name")
