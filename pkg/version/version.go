// Package version provides version information for the compiled binary, and an HTTP handler to serve the version information
// via an HTTP request.
package version

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/metricsstore"
)

var log = logger.New("version")

// BuildDate is the date when the binary was built
var BuildDate string

// GitCommit is the commit hash when the binary was built
var GitCommit string

// Version is the version of the compiled software
var Version string

// Info is a struct helpful for JSON serialization of the FSM Controller version information.
type Info struct {
	// Version is the version of the FSM Controller.
	Version string `json:"version,omitempty"`

	// GitCommit is the git commit hash of the FSM Controller.
	GitCommit string `json:"git_commit,omitempty"`

	// BuildDate is the build date of the FSM Controller.
	BuildDate string `json:"build_date,omitempty"`
}

// GetInfo returns the version info
func GetInfo() Info {
	return Info{
		Version:   Version,
		BuildDate: BuildDate,
		GitCommit: GitCommit,
	}
}

// GetVersionHandler returns an HTTP handler that returns the version info
func GetVersionHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		versionInfo := GetInfo()

		if jsonVersionInfo, err := json.Marshal(versionInfo); err != nil {
			log.Error().Err(err).Msgf("Error marshaling version info struct: %+v", versionInfo)
		} else {
			_, _ = fmt.Fprint(w, string(jsonVersionInfo))
		}
	})
}

// SetMetric is a one time call to expose the version info via prometheus metrics.
func SetMetric() {
	metricsstore.DefaultMetricsStore.VersionInfo.WithLabelValues(Version, BuildDate, GitCommit).Set(1)
}
