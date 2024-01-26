package v1

import (
	"strconv"

	"k8s.io/apimachinery/pkg/api/resource"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func percentage(gateway *gwv1.Gateway, annotation string, defVal int32) int32 {
	if len(gateway.Annotations) == 0 {
		return defVal
	}

	val, ok := gateway.Annotations[annotation]
	if !ok {
		return defVal
	}

	num, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		log.Error().Msgf("Failed to parse percentage %s: %s", val, err)
		return 1
	}

	if num < 0 || num > 100 {
		log.Error().Msgf("Invalid percentage %d, must be between 0 and 100", num)
		return 80
	}

	return int32(num)
}

func enabled(gateway *gwv1.Gateway, annotation string, defVal bool) bool {
	if len(gateway.Annotations) == 0 {
		return defVal
	}

	val, ok := gateway.Annotations[annotation]
	if !ok {
		return defVal
	}

	enabled, err := strconv.ParseBool(val)
	if err != nil {
		log.Error().Msgf("Failed to parse %s: %s", val, err)
		return false
	}

	return enabled
}

func replicas(gateway *gwv1.Gateway, annotation string, defVal int32) int32 {
	if len(gateway.Annotations) == 0 {
		return defVal
	}

	replicas, ok := gateway.Annotations[annotation]
	if !ok {
		return defVal
	}

	num, err := strconv.ParseInt(replicas, 10, 32)
	if err != nil {
		log.Error().Msgf("Failed to parse replicas %s: %s", replicas, err)
		return 1
	}

	return int32(num)
}

func resources(gateway *gwv1.Gateway, annotation string, defVal resource.Quantity) *resource.Quantity {
	if len(gateway.Annotations) == 0 {
		return &defVal
	}

	res, ok := gateway.Annotations[annotation]
	if !ok {
		return &defVal
	}

	q, err := resource.ParseQuantity(res)
	if err != nil {
		log.Error().Msgf("Failed to parse resource %s: %s", res, err)
		return &defVal
	}

	return &q
}

func hasTCP(gateway *gwv1.Gateway) bool {
	for _, listener := range gateway.Spec.Listeners {
		switch listener.Protocol {
		case gwv1.HTTPProtocolType, gwv1.TCPProtocolType, gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
			return true
		}
	}

	return false
}

func hasUDP(gateway *gwv1.Gateway) bool {
	for _, listener := range gateway.Spec.Listeners {
		if listener.Protocol == gwv1.UDPProtocolType {
			return true
		}
	}

	return false
}
