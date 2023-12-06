package ktoc

var (
	syncCloudNamespace string
	withGateway        bool
)

// SetSyncCloudNamespace sets sync namespace
func SetSyncCloudNamespace(ns string) {
	syncCloudNamespace = ns
}

// WithGateway sets enable or disable
func WithGateway(enable bool) {
	withGateway = enable
}
