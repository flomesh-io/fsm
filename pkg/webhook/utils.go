/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package webhook

import (
	"fmt"
	"net"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	netutils "k8s.io/utils/net"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// ValidateParentRefs validates the parent refs of gateway route resource
func ValidateParentRefs(refs []gwv1beta1.ParentReference) field.ErrorList {
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
func ValidateRouteHostnames(hostnames []gwv1beta1.Hostname) field.ErrorList {
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
