package ktoc

var (
	syncCloudNamespace string
	withGatewayAPI     bool
	withGatewayViaAddr string
	withGatewayViaPort int32
)

// SetSyncCloudNamespace sets sync namespace
func SetSyncCloudNamespace(ns string) {
	syncCloudNamespace = ns
}

// WithGatewayAPI sets enable or disable
func WithGatewayAPI(enable bool) {
	withGatewayAPI = enable
}

// WithGatewayViaAddr sets via addr
func WithGatewayViaAddr(addr string) {
	withGatewayViaAddr = addr
}

// WithGatewayViaPort sets via port
func WithGatewayViaPort(port int32) {
	withGatewayViaPort = port
}
