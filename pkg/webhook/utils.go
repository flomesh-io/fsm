package webhook

import (
	"fmt"
	"net"
	"strings"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	netutils "k8s.io/utils/net"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
)

// NewMutatingWebhookConfiguration creates a new MutatingWebhookConfiguration
func NewMutatingWebhookConfiguration(webhooks []admissionregv1.MutatingWebhook, meshName, fsmVersion string) *admissionregv1.MutatingWebhookConfiguration {
	if len(webhooks) == 0 {
		return nil
	}

	return &admissionregv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.DefaultMutatingWebhookConfigurationName,
			Labels: map[string]string{
				constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
				constants.FSMAppInstanceLabelKey: meshName,
				constants.FSMAppVersionLabelKey:  fsmVersion,
				constants.AppLabel:               constants.FSMControllerName,
			},
		},
		Webhooks: webhooks,
	}
}

// NewValidatingWebhookConfiguration creates a new ValidatingWebhookConfiguration
func NewValidatingWebhookConfiguration(webhooks []admissionregv1.ValidatingWebhook, meshName, fsmVersion string) *admissionregv1.ValidatingWebhookConfiguration {
	if len(webhooks) == 0 {
		return nil
	}

	return &admissionregv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.DefaultValidatingWebhookConfigurationName,
			Labels: map[string]string{
				constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
				constants.FSMAppInstanceLabelKey: meshName,
				constants.FSMAppVersionLabelKey:  fsmVersion,
				constants.AppLabel:               constants.FSMControllerName,
			},
		},
		Webhooks: webhooks,
	}
}

// ValidateParentRefs validates the parent refs of gateway route resource
func ValidateParentRefs(refs []gwv1.ParentReference) field.ErrorList {
	var errs field.ErrorList
	for i, ref := range refs {
		if ref.Port == nil {
			path := field.NewPath("spec").Child("parentRefs").Index(i)
			errs = append(errs, field.Required(path, "port must be set"))
		}
	}

	return errs
}

// IsValidHostname validates the hostname of gateway route resource
func IsValidHostname(hostname string) error {
	if net.ParseIP(hostname) != nil {
		return fmt.Errorf("invalid hostname %q: must be a DNS name, not an IP address", hostname)
	}

	if strings.Contains(hostname, "*") {
		if errs := validation.IsWildcardDNS1123Subdomain(hostname); errs != nil {
			return fmt.Errorf("invalid hostname %q: %v", hostname, errs)
		}
	} else {
		if errs := validation.IsDNS1123Subdomain(hostname); errs != nil {
			return fmt.Errorf("invalid hostname %q: %v", hostname, errs)
		}
	}

	return nil
}

// ValidateRouteHostnames validates the hostnames of gateway route resource
func ValidateRouteHostnames(hostnames []gwv1.Hostname) field.ErrorList {
	var errs field.ErrorList

	for i, hostname := range hostnames {
		h := string(hostname)
		if err := IsValidHostname(h); err != nil {
			path := field.NewPath("spec").
				Child("hostnames").Index(i)

			errs = append(errs, field.Invalid(path, h, fmt.Sprintf("%s", err)))
		}
	}

	return errs
}

// IsValidIPOrCIDR tests that the argument is a valid IP address.
func IsValidIPOrCIDR(value string) []string {
	if netutils.IsIPv4String(value) {
		return nil
	}

	if netutils.IsIPv6String(value) {
		return nil
	}

	if netutils.IsIPv4CIDRString(value) {
		return nil
	}

	if netutils.IsIPv6CIDRString(value) {
		return nil
	}

	return []string{"must be a valid IP address or CIDR, (e.g. 10.9.8.7 or 2001:db8::ffff or 192.0.2.0/24 or 2001:db8::/32)"}
}

func GetListenerIfHasMatchingPort(port gwv1.PortNumber, listeners []gwv1.Listener) *gwv1.Listener {
	if len(listeners) == 0 {
		return nil
	}

	for i, listener := range listeners {
		if port == listener.Port {
			return &listeners[i]
		}
	}

	return nil
}
