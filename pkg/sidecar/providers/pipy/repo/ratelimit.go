package repo

import "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"

// TCPRateLimit defines the rate limiting specification for
// the upstream host.
type TCPRateLimit struct {
	// Local specified the local rate limiting specification
	// for the upstream host.
	// Local rate limiting is enforced directly by the upstream
	// host without any involvement of a global rate limiting service.
	// This is applied as a token bucket rate limiter.
	// +optional
	Local *TCPLocalRateLimit `json:"Local,omitempty"`
}

// HTTPRateLimit defines the rate limiting specification for
// the upstream host.
type HTTPRateLimit struct {
	// Local specified the local rate limiting specification
	// for the upstream host.
	// Local rate limiting is enforced directly by the upstream
	// host without any involvement of a global rate limiting service.
	// This is applied as a token bucket rate limiter.
	// +optional
	Local *HTTPLocalRateLimit `json:"Local,omitempty"`
}

func newTCPRateLimit(spec *v1alpha1.LocalRateLimitSpec) *TCPRateLimit {
	if spec == nil || spec.TCP == nil {
		return nil
	}

	lrl := new(TCPRateLimit)
	lrl.Local = newTCPLocalRateLimit(spec.TCP)
	return lrl
}

func newHTTPRateLimit(spec *v1alpha1.LocalRateLimitSpec) *HTTPRateLimit {
	if spec == nil || spec.HTTP == nil {
		return nil
	}

	lrl := new(HTTPRateLimit)
	lrl.Local = newHTTPLocalRateLimit(spec.HTTP)
	return lrl
}

// TCPLocalRateLimit defines the local rate limiting specification
// for the upstream host at the TCP level.
type TCPLocalRateLimit struct {
	// Connections defines the number of connections allowed
	// per unit of time before rate limiting occurs.
	Connections uint32 `json:"Connections"`

	// StatTimeWindow specifies statistical time period of local rate limit
	StatTimeWindow float64 `json:"StatTimeWindow"`

	// Burst defines the number of connections above the baseline
	// rate that are allowed in a short period of time.
	// +optional
	Burst uint32 `json:"Burst,omitempty"`
}

func newTCPLocalRateLimit(spec *v1alpha1.TCPLocalRateLimitSpec) *TCPLocalRateLimit {
	if spec == nil {
		return nil
	}

	lrl := new(TCPLocalRateLimit)
	lrl.Connections = spec.Connections
	lrl.Burst = spec.Burst
	switch spec.Unit {
	case `second`:
		lrl.StatTimeWindow = 1
	case `minute`:
		lrl.StatTimeWindow = 60
	case `hour`:
		lrl.StatTimeWindow = 3600
	}
	return lrl
}

// HTTPLocalRateLimit defines the local rate limiting specification
// for the upstream host at the HTTP level.
type HTTPLocalRateLimit struct {
	// Requests defines the number of requests allowed
	// per unit of time before rate limiting occurs.
	Requests uint32 `json:"Requests"`

	// StatTimeWindow specifies statistical time period of local rate limit
	StatTimeWindow float64 `json:"StatTimeWindow"`

	// Burst defines the number of requests above the baseline
	// rate that are allowed in a short period of time.
	// +optional
	Burst uint32 `json:"Burst,omitempty"`

	// ResponseStatusCode defines the HTTP status code to use for responses
	// to rate limited requests. Code must be in the 400-599 (inclusive)
	// error range. If not specified, a default of 429 (Too Many Requests) is used.
	// +optional
	ResponseStatusCode uint32 `json:"ResponseStatusCode,omitempty"`

	// ResponseHeadersToAdd defines the list of HTTP headers that should be
	// added to each response for requests that have been rate limited.
	// +optional
	ResponseHeadersToAdd []HTTPHeaderValue `json:"ResponseHeadersToAdd,omitempty"`
}

func newHTTPLocalRateLimit(spec *v1alpha1.HTTPLocalRateLimitSpec) *HTTPLocalRateLimit {
	if spec == nil {
		return nil
	}

	lrl := new(HTTPLocalRateLimit)
	lrl.Requests = spec.Requests
	lrl.Burst = spec.Burst
	lrl.ResponseStatusCode = spec.ResponseStatusCode
	if len(spec.ResponseHeadersToAdd) > 0 {
		for _, header := range spec.ResponseHeadersToAdd {
			lrl.ResponseHeadersToAdd = append(lrl.ResponseHeadersToAdd, newHTTPHeaderValue(header))
		}
	}
	switch spec.Unit {
	case `second`:
		lrl.StatTimeWindow = 1
	case `minute`:
		lrl.StatTimeWindow = 60
	case `hour`:
		lrl.StatTimeWindow = 3600
	}
	return lrl
}

// HTTPHeaderValue defines an HTTP header name/value pair
type HTTPHeaderValue struct {
	// Name defines the name of the HTTP header.
	Name string `json:"Name"`

	// Value defines the value of the header corresponding to the name key.
	Value string `json:"Value"`
}

func newHTTPHeaderValue(header v1alpha1.HTTPHeaderValue) HTTPHeaderValue {
	return HTTPHeaderValue{
		Name:  header.Name,
		Value: header.Value,
	}
}

// HTTPPerRouteRateLimit defines the rate limiting specification
// per HTTP route.
type HTTPPerRouteRateLimit struct {
	// Local defines the local rate limiting specification
	// applied per HTTP route.
	Local *HTTPLocalRateLimit `json:"Local,omitempty"`
}

func newHTTPPerRouteRateLimit(spec *v1alpha1.HTTPPerRouteRateLimitSpec) *HTTPPerRouteRateLimit {
	if spec == nil {
		return nil
	}

	rrl := new(HTTPPerRouteRateLimit)
	rrl.Local = newHTTPLocalRateLimit(spec.Local)
	return rrl
}
