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

package cache

import (
	"context"
	"fmt"
	apicommons "github.com/flomesh-io/fsm/apis"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/certificate/utils"
	"github.com/flomesh-io/fsm/pkg/commons"
	ingresspipy "github.com/flomesh-io/fsm/pkg/ingress"
	"github.com/flomesh-io/fsm/pkg/ingress/controller"
	"github.com/flomesh-io/fsm/pkg/kube"
	repocfg "github.com/flomesh-io/fsm/pkg/route"
	"github.com/flomesh-io/fsm/pkg/util"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/tools/events"
	"k8s.io/klog/v2"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type BaseIngressInfo struct {
	headers        map[string]string
	host           string
	path           string
	backend        ServicePortName
	rewrite        []string // rewrite in format: ["^/flomesh/?", "/"],  first element is from, second is to
	sessionSticky  bool
	lbType         apicommons.AlgoBalancer
	upstream       *repocfg.UpstreamSpec
	certificate    *repocfg.CertificateSpec
	isTLS          bool
	isWildcardHost bool
	verifyClient   bool
	verifyDepth    int
	trustedCA      *repocfg.CertificateSpec
}

var _ Route = &BaseIngressInfo{}

func (info BaseIngressInfo) String() string {
	return fmt.Sprintf("%s%s", info.host, info.path)
}

func (info BaseIngressInfo) Headers() map[string]string {
	return info.headers
}

func (info BaseIngressInfo) Host() string {
	return info.host
}

func (info BaseIngressInfo) Path() string {
	return info.path
}

func (info BaseIngressInfo) Backend() ServicePortName {
	return info.backend
}

func (info BaseIngressInfo) Rewrite() []string {
	return info.rewrite
}

func (info BaseIngressInfo) SessionSticky() bool {
	return info.sessionSticky
}

func (info BaseIngressInfo) LBType() apicommons.AlgoBalancer {
	return info.lbType
}

func (info BaseIngressInfo) UpstreamSSLName() string {
	if info.upstream == nil {
		return ""
	}

	return info.upstream.SSLName
}

func (info BaseIngressInfo) UpstreamSSLCert() *repocfg.CertificateSpec {
	if info.upstream == nil {
		return nil
	}

	return info.upstream.SSLCert
}

func (info BaseIngressInfo) UpstreamSSLVerify() bool {
	if info.upstream == nil {
		return false
	}

	return info.upstream.SSLVerify
}

func (info BaseIngressInfo) Certificate() *repocfg.CertificateSpec {
	return info.certificate
}

func (info BaseIngressInfo) IsTLS() bool {
	return info.isTLS
}

func (info BaseIngressInfo) IsWildcardHost() bool {
	return info.isWildcardHost
}

func (info BaseIngressInfo) VerifyClient() bool {
	return info.verifyClient
}

func (info BaseIngressInfo) VerifyDepth() int {
	return info.verifyDepth
}

func (info BaseIngressInfo) TrustedCA() *repocfg.CertificateSpec {
	return info.trustedCA
}

func (info BaseIngressInfo) Protocol() string {
	if info.upstream == nil {
		return ""
	}

	return info.upstream.Protocol
}

type IngressMap map[RouteKey]Route

type RouteKey struct {
	ServicePortName
	Host string
	Path string
}

func (irk *RouteKey) String() string {
	return fmt.Sprintf("%s#%s#%s", irk.Host, irk.Path, irk.ServicePortName.String())
}

type ingressChange struct {
	previous IngressMap
	current  IngressMap
}

type IngressChangeTracker struct {
	lock        sync.Mutex
	items       map[types.NamespacedName]*ingressChange
	controllers *controller.Controllers
	k8sAPI      *kube.K8sAPI
	recorder    events.EventRecorder
}

func NewIngressChangeTracker(k8sAPI *kube.K8sAPI, controllers *controller.Controllers, recorder events.EventRecorder, mgr certificate.Manager) *IngressChangeTracker {
	return &IngressChangeTracker{
		items:       make(map[types.NamespacedName]*ingressChange),
		controllers: controllers,
		k8sAPI:      k8sAPI,
		recorder:    recorder,
	}
}

func (ict *IngressChangeTracker) newBaseIngressInfo(rule networkingv1.IngressRule, path networkingv1.HTTPIngressPath, svcPortName ServicePortName) *BaseIngressInfo {
	switch *path.PathType {
	case networkingv1.PathTypeExact:
		return &BaseIngressInfo{
			headers:        make(map[string]string),
			host:           rule.Host,
			path:           path.Path,
			backend:        svcPortName,
			isWildcardHost: isWildcardHost(rule.Host),
		}
	case networkingv1.PathTypePrefix:
		var hostPath string
		if strings.HasSuffix(path.Path, "/*") {
			hostPath = path.Path
		} else {
			if strings.HasSuffix(path.Path, "/") {
				hostPath = path.Path + "*"
			} else {
				hostPath = path.Path + "/*"
			}
		}

		return &BaseIngressInfo{
			headers:        make(map[string]string),
			host:           rule.Host,
			path:           hostPath,
			backend:        svcPortName,
			isWildcardHost: isWildcardHost(rule.Host),
		}
	default:
		return nil
	}
}

func isWildcardHost(host string) bool {
	if host != "" {
		if errs := validation.IsWildcardDNS1123Subdomain(host); len(errs) == 0 {
			return true
		}
	}

	return false
}

func (ict *IngressChangeTracker) Update(previous, current *networkingv1.Ingress) bool {
	ing := current
	if ing == nil {
		ing = previous
	}

	if ing == nil {
		return false
	}

	if !ingresspipy.IsValidPipyIngress(ing) {
		return false
	}

	namespacedName := types.NamespacedName{Namespace: ing.Namespace, Name: ing.Name}

	ict.lock.Lock()
	defer ict.lock.Unlock()

	change, exists := ict.items[namespacedName]
	if !exists {
		change = &ingressChange{}
		change.previous = ict.ingressToIngressMap(previous)
		ict.items[namespacedName] = change
	}
	change.current = ict.ingressToIngressMap(current)

	if reflect.DeepEqual(change.previous, change.current) {
		delete(ict.items, namespacedName)
	} else {
		klog.V(2).Infof("Ingress %s updated: %d rules", namespacedName, len(change.current))
	}

	return len(ict.items) > 0
}

func (ict *IngressChangeTracker) ingressToIngressMap(ing *networkingv1.Ingress) IngressMap {
	if ing == nil {
		return nil
	}

	ingressMap := make(IngressMap)
	ingKey := kube.MetaNamespaceKey(ing)

	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}

		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				// skip non-service backends
				klog.V(3).Infof("Ingress %q and path %q does not contain a service backend", ingKey, path.Path)
				continue
			}

			svcPortName := ict.servicePortName(ing.Namespace, path.Backend.Service)
			// in case of error or unexpected condition, ignore it
			if svcPortName == nil {
				klog.Warningf("svcPortName is nil for Namespace: %q,  Path: %v", ing.Namespace, path)
				continue
			}
			klog.V(5).Infof("ServicePortName %q", svcPortName.String())

			baseIngInfo := ict.newBaseIngressInfo(rule, path, *svcPortName)
			if baseIngInfo == nil {
				continue
			}

			routeKey := RouteKey{
				ServicePortName: *svcPortName,
				Host:            baseIngInfo.Host(),
				Path:            baseIngInfo.Path(),
			}

			// already exists, first one wins
			if _, ok := ingressMap[routeKey]; ok {
				klog.Warningf("Duplicate route for tuple: %q", routeKey.String())
				continue
			}

			ingressMap[routeKey] = ict.enrichIngressInfo(&rule, ing, baseIngInfo)

			klog.V(5).Infof("Route %q is linked to rule %v", routeKey.String(), ingressMap[routeKey])
		}
	}

	return ingressMap
}

func (ict *IngressChangeTracker) servicePortName(namespace string, service *networkingv1.IngressServiceBackend) *ServicePortName {
	if service != nil {
		if service.Port.Name != "" {
			return createSvcPortNameInstance(namespace, service.Name, service.Port.Name)
		}

		if service.Port.Number > 0 {
			namespacedSvcName := types.NamespacedName{
				Namespace: namespace,
				Name:      service.Name,
			}

			svc, err := ict.findService(namespace, service)
			if err != nil {
				klog.Errorf("Not able to find service %s from anywhere, %v", namespacedSvcName.String(), err)
				return nil
			}

			for _, port := range svc.Spec.Ports {
				if port.Port == service.Port.Number {
					return createSvcPortNameInstance(namespace, service.Name, port.Name)
				}
			}
		}

	}

	return nil
}

func createSvcPortNameInstance(namespace, serviceName, portName string) *ServicePortName {
	return &ServicePortName{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      serviceName,
		},
		Port: portName,
		// Ingress so far can only handle TCP
		Protocol: corev1.ProtocolTCP,
	}
}

// svcName in namespace/name format
func (ict *IngressChangeTracker) findService(namespace string, service *networkingv1.IngressServiceBackend) (*corev1.Service, error) {
	svcName := fmt.Sprintf("%s/%s", namespace, service.Name)

	// first, find in local store
	svc, exists, err := ict.controllers.Service.Store.GetByKey(svcName)
	if err != nil {
		return nil, err
	}
	if !exists {
		klog.Warningf("no object matching key %q in local store, will try to retrieve it from API server.", svcName)
		// if not exists in local, retrieve it from remote API server, this's Plan-B, should seldom happns
		svc, err = ict.k8sAPI.Client.CoreV1().Services(namespace).Get(context.TODO(), service.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		klog.V(5).Infof("Found service %q from API server.", svcName)
	} else {
		klog.V(5).Infof("Found service %q in local store.", svcName)
	}
	return svc.(*corev1.Service), nil
}

func (ict *IngressChangeTracker) checkoutChanges() []*ingressChange {
	ict.lock.Lock()
	defer ict.lock.Unlock()

	changes := []*ingressChange{}
	for _, change := range ict.items {
		changes = append(changes, change)
	}
	ict.items = make(map[types.NamespacedName]*ingressChange)
	return changes
}

func (im IngressMap) Update(changes *IngressChangeTracker) {
	im.apply(changes)
}

func (im IngressMap) apply(ict *IngressChangeTracker) {
	if ict == nil {
		return
	}

	changes := ict.checkoutChanges()
	for _, change := range changes {
		im.unmerge(change.previous)
		im.merge(change.current)
	}
}

func (im IngressMap) merge(other IngressMap) {
	for svcPortName := range other {
		im[svcPortName] = other[svcPortName]
	}
}

func (im IngressMap) unmerge(other IngressMap) {
	for svcPortName := range other {
		delete(im, svcPortName)
	}
}

// enrichIngressInfo is for extending K8s standard ingress
func (ict *IngressChangeTracker) enrichIngressInfo(rule *networkingv1.IngressRule, ing *networkingv1.Ingress, info *BaseIngressInfo) Route {
	if len(ing.Spec.TLS) > 0 {
		info.isTLS = true

		secretName := ict.getTLSSecretName(rule, ing)
		klog.V(5).Infof("secret name = %q ...", secretName)
		if secretName != "" {
			cert := ict.fetchSSLCert(ing, ing.Namespace, secretName)

			if cert != nil && cert.Cert != "" && cert.Key != "" {
				klog.V(5).Infof("Found certificate for host %q from secret %s/%s", rule.Host, ing.Namespace, secretName)
				info.certificate = cert
			}
		}
	}

	if ing.Annotations == nil {
		klog.Warningf("Ingress %s/%s doesn't have any annotations", ing.Namespace, ing.Name)
		return info
	}

	klog.V(5).Infof("Annotations of Ingress %s/%s: %v", ing.Namespace, ing.Name, ing.Annotations)

	// enrich rewrite if exists
	rewriteFrom := ing.Annotations[ingresspipy.PipyIngressAnnotationRewriteFrom]
	rewriteTo := ing.Annotations[ingresspipy.PipyIngressAnnotationRewriteTo]
	if rewriteFrom != "" && rewriteTo != "" {
		info.rewrite = []string{rewriteFrom, rewriteTo}
	}

	// enrich session sticky
	sticky := ing.Annotations[ingresspipy.PipyIngressAnnotationSessionSticky]
	switch strings.ToLower(sticky) {
	case "yes", "true", "1", "on":
		info.sessionSticky = true
	case "no", "false", "0", "off", "":
		info.sessionSticky = false
	default:
		klog.Warningf("Invalid value %q of annotation pipy.ingress.kubernetes.io/session-sticky of Ingress %s/%s, setting session sticky to false", sticky, ing.Namespace, ing.Name)
		info.sessionSticky = false
	}

	// enrich LB type
	lbValue := ing.Annotations[ingresspipy.PipyIngressAnnotationLoadBalancer]
	if lbValue == "" {
		lbValue = string(apicommons.RoundRobinLoadBalancer)
	}

	balancer := apicommons.AlgoBalancer(lbValue)
	switch balancer {
	case apicommons.RoundRobinLoadBalancer, apicommons.LeastWorkLoadBalancer, apicommons.HashingLoadBalancer:
		info.lbType = balancer
	default:
		klog.Errorf("%q is ignored, as it's not a supported Load Balancer type, uses default RoundRobinLoadBalancer.", lbValue)
		info.lbType = apicommons.RoundRobinLoadBalancer
	}

	// Upstream SNI
	upstreamSSLName := ing.Annotations[ingresspipy.PipyIngressAnnotationUpstreamSSLName]
	if upstreamSSLName != "" {
		if info.upstream == nil {
			info.upstream = &repocfg.UpstreamSpec{}
		}
		info.upstream.SSLName = upstreamSSLName
	}

	// Upstream SSL Secret
	upstreamSSLSecret := ing.Annotations[ingresspipy.PipyIngressAnnotationUpstreamSSLSecret]
	if upstreamSSLSecret != "" {

		ns, name, err := util.SecretNamespaceAndName(upstreamSSLSecret, ing)
		if err == nil {
			if info.upstream == nil {
				info.upstream = &repocfg.UpstreamSpec{}
			}
			info.upstream.SSLCert = ict.fetchSSLCert(ing, ns, name)
		} else {
			klog.Errorf("Invalid value %q of annotation pipy.ingress.kubernetes.io/upstream-ssl-secret of Ingress %s/%s: %s", upstreamSSLSecret, ing.Namespace, ing.Name, err)
		}
	}

	// Upstream SSL Verify
	upstreamSSLVerify := ing.Annotations[ingresspipy.PipyIngressAnnotationUpstreamSSLVerify]
	if info.upstream == nil {
		info.upstream = &repocfg.UpstreamSpec{}
	}
	switch strings.ToLower(upstreamSSLVerify) {
	case "yes", "true", "1", "on":
		info.upstream.SSLVerify = true
	case "no", "false", "0", "off", "":
		info.upstream.SSLVerify = false
	default:
		klog.Warningf("Invalid value %q of annotation pipy.ingress.kubernetes.io/upstream-ssl-verify of Ingress %s/%s, setting upstream-ssl-verify to false", upstreamSSLVerify, ing.Namespace, ing.Name)
		info.upstream.SSLVerify = false
	}

	// Verify Client
	verifyClient := ing.Annotations[ingresspipy.PipyIngressAnnotationTLSVerifyClient]
	switch strings.ToLower(verifyClient) {
	case "yes", "true", "1", "on":
		info.verifyClient = true
	case "no", "false", "0", "off", "":
		info.verifyClient = false
	default:
		klog.Warningf("Invalid value %q of annotation pipy.ingress.kubernetes.io/tls-verify-client of Ingress %s/%s, setting verify client to false", verifyClient, ing.Namespace, ing.Name)
		info.verifyClient = false
	}

	// Verify Depth
	verifyDepth := ing.Annotations[ingresspipy.PipyIngressAnnotationTLSVerifyDepth]
	if verifyDepth == "" {
		verifyDepth = "1"
	}
	depth, err := strconv.Atoi(verifyDepth)
	if err == nil {
		info.verifyDepth = depth
	} else {
		klog.Warningf("Invalid value %q of annotation pipy.ingress.kubernetes.io/tls-verify-depth of Ingress %s/%s, setting verify depth to 1", ing.Annotations[ingresspipy.PipyIngressAnnotationTLSVerifyDepth], ing.Namespace, ing.Name)
		info.verifyDepth = 1
	}

	// Trusted CA
	if info.certificate != nil && info.certificate.CA != "" {
		info.trustedCA = info.certificate
	}
	trustedCASecret := ing.Annotations[ingresspipy.PipyIngressAnnotationTLSTrustedCASecret]
	if trustedCASecret != "" {
		ns, name, err := util.SecretNamespaceAndName(trustedCASecret, ing)
		if err == nil {
			info.trustedCA = ict.fetchSSLCert(ing, ns, name)
		} else {
			klog.Errorf("Invalid value %q of annotation pipy.ingress.kubernetes.io/tls-trusted-ca-secret of Ingress %s/%s: %s", trustedCASecret, ing.Namespace, ing.Name, err)
		}
	}

	// Backend Protocol
	backendProtocol := strings.ToUpper(ing.Annotations[ingresspipy.PipyIngressAnnotationBackendProtocol])
	if info.upstream == nil {
		info.upstream = &repocfg.UpstreamSpec{}
	}
	switch backendProtocol {
	case "GRPC":
		info.upstream.Protocol = "GRPC"
		//default:
		//    info.upstream.Protocol = "HTTP"
	}

	return info
}

func (ict *IngressChangeTracker) getTLSSecretName(rule *networkingv1.IngressRule, ing *networkingv1.Ingress) string {
	host := rule.Host
	lowercaseHost := strings.ToLower(host)
	for _, tls := range ing.Spec.TLS {
		for _, tlsHost := range tls.Hosts {
			if lowercaseHost == strings.ToLower(tlsHost) {
				return tls.SecretName
			}
		}
	}

	for _, tls := range ing.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}

		cert := ict.fetchSSLCert(ing, ing.Namespace, tls.SecretName)
		if cert == nil {
			continue
		} else {
			if cert.Cert == "" || cert.Key == "" {
				klog.Warningf("Empty Certificate/PrivateKey from secret %s/%s", ing.Namespace, tls.SecretName)
				continue
			}

			x509Cert, err := utils.ConvertPEMCertToX509([]byte(cert.Cert))
			if err != nil {
				klog.Warningf("Failed to convert PEM cert to X509: %s", err)
				continue
			}

			if err := x509Cert.VerifyHostname(host); err != nil {
				klog.Warningf("Failed validating SSL certificate %s/%s for host %q: %v", ing.Namespace, tls.SecretName, host, err)
				continue
			}

			klog.V(5).Infof("Found SSL certificate matching host %q: %s/%s", host, ing.Namespace, tls.SecretName)
			return tls.SecretName
		}
	}

	return ""
}

func (ict *IngressChangeTracker) fetchSSLCert(ing *networkingv1.Ingress, ns, name string) *repocfg.CertificateSpec {
	if ns == "" {
		klog.Warningf("namespace is empty, will use Ingress's namespace")
		ns = ing.Namespace
	}

	if name == "" {
		klog.Errorf("Secret name is empty of Ingress %s/%s", ing.Namespace, ing.Name)
		return nil
	}

	klog.V(5).Infof("Fetching secret %s/%s ...", ns, name)
	secret, err := ict.controllers.Secret.Lister.Secrets(ns).Get(name)

	if err != nil {
		klog.Errorf("Failed to get secret %s/%s of Ingress %s/%s: %s", ns, name, ing.Namespace, ing.Name, err)
		return nil
	}

	return &repocfg.CertificateSpec{
		Cert: string(secret.Data[commons.TLSCertName]),
		Key:  string(secret.Data[commons.TLSPrivateKeyName]),
		CA:   string(secret.Data[commons.RootCACertName]),
	}
}
