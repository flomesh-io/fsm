package ktoc

var (
	syncCloudNamespace string

	withGatewayIngress     bool
	withGatewayIngressAddr string
	withGatewayIngressPort int32
)

// SetSyncCloudNamespace sets sync namespace
func SetSyncCloudNamespace(ns string) {
	syncCloudNamespace = ns
}

// WithGatewayIngress sets enable or disable
func WithGatewayIngress(enable bool) {
	withGatewayIngress = enable
}

// WithGatewayIngressAddr sets via addr
func WithGatewayIngressAddr(addr string) {
	withGatewayIngressAddr = addr
}

// WithGatewayIngressPort sets via port
func WithGatewayIngressPort(port int32) {
	withGatewayIngressPort = port
}
