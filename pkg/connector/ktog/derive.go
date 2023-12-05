package ktog

var (
	withGatewayIngressHTTPPort int32
	withGatewayIngressGRPCPort int32

	withGatewayEgressHTTPPort int32
	withGatewayEgressGRPCPort int32
)

// WithGatewayIngressHTTPPort sets ingress http port
func WithGatewayIngressHTTPPort(port int32) {
	withGatewayIngressHTTPPort = port
}

// WithGatewayIngressGRPCPort sets ingress grpc port
func WithGatewayIngressGRPCPort(port int32) {
	withGatewayIngressGRPCPort = port
}

// WithGatewayEgressHTTPPort sets egress http port
func WithGatewayEgressHTTPPort(port int32) {
	withGatewayEgressHTTPPort = port
}

// WithGatewayEgressGRPCPort sets egress grpc port
func WithGatewayEgressGRPCPort(port int32) {
	withGatewayEgressGRPCPort = port
}
