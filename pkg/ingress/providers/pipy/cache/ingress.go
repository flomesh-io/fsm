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
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"strconv"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/events"

	apiconstants "github.com/flomesh-io/fsm/pkg/apis"
	"github.com/flomesh-io/fsm/pkg/constants"
	repocfg "github.com/flomesh-io/fsm/pkg/ingress/providers/pipy/route"
	ingresspipy "github.com/flomesh-io/fsm/pkg/ingress/providers/pipy/utils"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/utils"
)

type baseIngressInfo struct {
	headers        map[string]string
	host           string
	path           string
	backend        ServicePortName
	rewrite        []string // rewrite in format: ["^/flomesh/?", "/"],  first element is from, second is to
	sessionSticky  bool
	lbType         apiconstants.AlgoBalancer
	upstream       *repocfg.UpstreamSpec
	certificate    *repocfg.CertificateSpec
	isTLS          bool
	isWildcardHost bool
	verifyClient   bool
	verifyDepth    int
	trustedCA      *repocfg.CertificateSpec
}

var _ Route = &baseIngressInfo{}

func (info baseIngressInfo) String() string {
	return fmt.Sprintf("%s%s", info.host, info.path)
}

func (info baseIngressInfo) Headers() map[string]string {
	return info.headers
}

func (info baseIngressInfo) Host() string {
	return info.host
}

func (info baseIngressInfo) Path() string {
	return info.path
}

func (info baseIngressInfo) Backend() ServicePortName {
	return info.backend
}

func (info baseIngressInfo) Rewrite() []string {
	return info.rewrite
}

func (info baseIngressInfo) SessionSticky() bool {
	return info.sessionSticky
}

func (info baseIngressInfo) LBType() apiconstants.AlgoBalancer {
	return info.lbType
}

func (info baseIngressInfo) UpstreamSSLName() string {
	if info.upstream == nil {
		return ""
	}

	return info.upstream.SSLName
}

func (info baseIngressInfo) UpstreamSSLCert() *repocfg.CertificateSpec {
	if info.upstream == nil {
		return nil
	}

	return info.upstream.SSLCert
}

func (info baseIngressInfo) UpstreamSSLVerify() bool {
	if info.upstream == nil {
		return false
	}

	return info.upstream.SSLVerify
}

func (info baseIngressInfo) Certificate() *repocfg.CertificateSpec {
	return info.certificate
}

func (info baseIngressInfo) IsTLS() bool {
	return info.isTLS
}

func (info baseIngressInfo) IsWildcardHost() bool {
	return info.isWildcardHost
}

func (info baseIngressInfo) VerifyClient() bool {
	return info.verifyClient
}

func (info baseIngressInfo) VerifyDepth() int {
	return info.verifyDepth
}

func (info baseIngressInfo) TrustedCA() *repocfg.CertificateSpec {
	return info.trustedCA
}

func (info baseIngressInfo) Protocol() string {
	if info.upstream == nil {
		return ""
	}

	return info.upstream.Protocol
}

// IngressMap is a map of Ingresses
type IngressMap map[RouteKey]Route

// RouteKey is a key for IngressMap
type RouteKey struct {
	ServicePortName
	Host string
	Path string
}

// String returns a string representation of RouteKey
func (irk *RouteKey) String() string {
	return fmt.Sprintf("%s#%s#%s", irk.Host, irk.Path, irk.ServicePortName.String())
}

type ingressChange struct {
	previous IngressMap
	current  IngressMap
}

// IngressChangeTracker tracks changes to Ingresses
type IngressChangeTracker struct {
	lock       sync.Mutex
	items      map[types.NamespacedName]*ingressChange
	kubeClient kubernetes.Interface
	informers  *fsminformers.InformerCollection
	recorder   events.EventRecorder
	client     cache.Cache
}

// NewIngressChangeTracker creates a new IngressChangeTracker
func NewIngressChangeTracker(client cache.Cache, recorder events.EventRecorder) *IngressChangeTracker {
	return &IngressChangeTracker{
		items:    make(map[types.NamespacedName]*ingressChange),
		recorder: recorder,
		client:   client,
	}
}

func (t *IngressChangeTracker) newBaseIngressInfo(rule networkingv1.IngressRule, path networkingv1.HTTPIngressPath, svcPortName ServicePortName) *baseIngressInfo {
	switch *path.PathType {
	case networkingv1.PathTypeExact:
		return &baseIngressInfo{
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

		return &baseIngressInfo{
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

// Update updates the tracker with the given Ingresses
func (t *IngressChangeTracker) Update(previous, current *networkingv1.Ingress) bool {
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

	t.lock.Lock()
	defer t.lock.Unlock()

	change, exists := t.items[namespacedName]
	if !exists {
		change = &ingressChange{}
		change.previous = t.ingressToIngressMap(previous)
		t.items[namespacedName] = change
	}
	change.current = t.ingressToIngressMap(current)

	if reflect.DeepEqual(change.previous, change.current) {
		delete(t.items, namespacedName)
	} else {
		log.Info().Msgf("Ingress %s updated: %d rules", namespacedName, len(change.current))
	}

	return len(t.items) > 0
}

func (t *IngressChangeTracker) ingressToIngressMap(ing *networkingv1.Ingress) IngressMap {
	if ing == nil {
		return nil
	}

	ingressMap := make(IngressMap)
	ingKey := ingresspipy.MetaNamespaceKey(ing)

	for _, rule := range ing.Spec.Rules {
		rule := rule // fix lint GO-LOOP-REF
		if rule.HTTP == nil {
			continue
		}

		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				// skip non-service backends
				log.Info().Msgf("Ingress %q and path %q does not contain a service backend", ingKey, path.Path)
				continue
			}

			svcPortName := t.servicePortName(ing.Namespace, path.Backend.Service)
			// in case of error or unexpected condition, ignore it
			if svcPortName == nil {
				log.Warn().Msgf("svcPortName is nil for Namespace: %q,  Path: %v", ing.Namespace, path)
				continue
			}
			log.Info().Msgf("ServicePortName %q", svcPortName.String())

			baseIngInfo := t.newBaseIngressInfo(rule, path, *svcPortName)
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
				log.Warn().Msgf("Duplicate route for tuple: %q", routeKey.String())
				continue
			}

			ingressMap[routeKey] = t.enrichIngressInfo(&rule, ing, baseIngInfo)

			log.Info().Msgf("Route %q is linked to rule %v", routeKey.String(), ingressMap[routeKey])
		}
	}

	return ingressMap
}

func (t *IngressChangeTracker) servicePortName(namespace string, service *networkingv1.IngressServiceBackend) *ServicePortName {
	if service != nil {
		if service.Port.Name != "" {
			return createSvcPortNameInstance(namespace, service.Name, service.Port.Name)
		}

		if service.Port.Number > 0 {
			namespacedSvcName := types.NamespacedName{
				Namespace: namespace,
				Name:      service.Name,
			}

			svc, err := t.findService(namespace, service)
			if err != nil {
				log.Error().Msgf("Not able to find service %s from anywhere, %v", namespacedSvcName.String(), err)
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
func (t *IngressChangeTracker) findService(namespace string, service *networkingv1.IngressServiceBackend) (*corev1.Service, error) {
	svcName := fmt.Sprintf("%s/%s", namespace, service.Name)

	// first, find in local store
	svc, exists, err := t.informers.GetByKey(fsminformers.InformerKeyService, svcName)
	if err != nil {
		return nil, err
	}
	if !exists {
		log.Warn().Msgf("no object matching key %q in local store, will try to retrieve it from API server.", svcName)
		// if not exists in local, retrieve it from remote API server, this's Plan-B, should seldom happns
		svc, err = t.kubeClient.CoreV1().Services(namespace).Get(context.TODO(), service.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		log.Info().Msgf("Found service %q from API server.", svcName)
	} else {
		log.Info().Msgf("Found service %q in local store.", svcName)
	}
	return svc.(*corev1.Service), nil
}

func (t *IngressChangeTracker) checkoutChanges() []*ingressChange {
	t.lock.Lock()
	defer t.lock.Unlock()

	changes := []*ingressChange{}
	for _, change := range t.items {
		changes = append(changes, change)
	}
	t.items = make(map[types.NamespacedName]*ingressChange)
	return changes
}

// Update updates the ingress map with the changes
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
func (t *IngressChangeTracker) enrichIngressInfo(rule *networkingv1.IngressRule, ing *networkingv1.Ingress, info *baseIngressInfo) Route {
	if len(ing.Spec.TLS) > 0 {
		info.isTLS = true

		secretName := t.getTLSSecretName(rule, ing)
		log.Info().Msgf("secret name = %q ...", secretName)
		if secretName != "" {
			cert := t.fetchSSLCert(ing, ing.Namespace, secretName)

			if cert != nil && cert.Cert != "" && cert.Key != "" {
				log.Info().Msgf("Found certificate for host %q from secret %s/%s", rule.Host, ing.Namespace, secretName)
				info.certificate = cert
			}
		}
	}

	if ing.Annotations == nil {
		log.Warn().Msgf("Ingress %s/%s doesn't have any annotations", ing.Namespace, ing.Name)
		return info
	}

	log.Info().Msgf("Annotations of Ingress %s/%s: %v", ing.Namespace, ing.Name, ing.Annotations)

	// enrich rewrite if exists
	rewriteFrom := ing.Annotations[constants.PipyIngressAnnotationRewriteFrom]
	rewriteTo := ing.Annotations[constants.PipyIngressAnnotationRewriteTo]
	if rewriteFrom != "" && rewriteTo != "" {
		info.rewrite = []string{rewriteFrom, rewriteTo}
	}

	// enrich session sticky
	sticky := ing.Annotations[constants.PipyIngressAnnotationSessionSticky]
	info.sessionSticky = utils.ParseEnabled(sticky)

	// enrich LB type
	lbValue := ing.Annotations[constants.PipyIngressAnnotationLoadBalancer]
	if lbValue == "" {
		lbValue = string(apiconstants.RoundRobinLoadBalancer)
	}

	balancer := apiconstants.AlgoBalancer(lbValue)
	switch balancer {
	case apiconstants.RoundRobinLoadBalancer, apiconstants.LeastWorkLoadBalancer, apiconstants.HashingLoadBalancer:
		info.lbType = balancer
	default:
		log.Error().Msgf("%q is ignored, as it's not a supported Load Balancer type, uses default RoundRobinLoadBalancer.", lbValue)
		info.lbType = apiconstants.RoundRobinLoadBalancer
	}

	// Upstream SNI
	upstreamSSLName := ing.Annotations[constants.PipyIngressAnnotationUpstreamSSLName]
	if upstreamSSLName != "" {
		if info.upstream == nil {
			info.upstream = &repocfg.UpstreamSpec{}
		}
		info.upstream.SSLName = upstreamSSLName
	}

	// Upstream SSL Secret
	upstreamSSLSecret := ing.Annotations[constants.PipyIngressAnnotationUpstreamSSLSecret]
	if upstreamSSLSecret != "" {
		ns, name, err := utils.SecretNamespaceAndName(upstreamSSLSecret, ing)
		if err == nil {
			if info.upstream == nil {
				info.upstream = &repocfg.UpstreamSpec{}
			}
			info.upstream.SSLCert = t.fetchSSLCert(ing, ns, name)
		} else {
			log.Error().Msgf("Invalid value %q of annotation pipy.ingress.kubernetes.io/upstream-ssl-secret of Ingress %s/%s: %s", upstreamSSLSecret, ing.Namespace, ing.Name, err)
		}
	}

	// Upstream SSL Verify
	upstreamSSLVerify := ing.Annotations[constants.PipyIngressAnnotationUpstreamSSLVerify]
	if info.upstream == nil {
		info.upstream = &repocfg.UpstreamSpec{}
	}
	info.upstream.SSLVerify = utils.ParseEnabled(upstreamSSLVerify)

	// Verify Client
	verifyClient := ing.Annotations[constants.PipyIngressAnnotationTLSVerifyClient]
	info.verifyClient = utils.ParseEnabled(verifyClient)

	// Verify Depth
	verifyDepth := ing.Annotations[constants.PipyIngressAnnotationTLSVerifyDepth]
	if verifyDepth == "" {
		verifyDepth = "1"
	}
	depth, err := strconv.Atoi(verifyDepth)
	if err == nil {
		info.verifyDepth = depth
	} else {
		log.Warn().Msgf("Invalid value %q of annotation pipy.ingress.kubernetes.io/tls-verify-depth of Ingress %s/%s, setting verify depth to 1", ing.Annotations[constants.PipyIngressAnnotationTLSVerifyDepth], ing.Namespace, ing.Name)
		info.verifyDepth = 1
	}

	// Trusted CA
	if info.certificate != nil && info.certificate.CA != "" {
		info.trustedCA = info.certificate
	}
	trustedCASecret := ing.Annotations[constants.PipyIngressAnnotationTLSTrustedCASecret]
	if trustedCASecret != "" {
		ns, name, err := utils.SecretNamespaceAndName(trustedCASecret, ing)
		if err == nil {
			info.trustedCA = t.fetchSSLCert(ing, ns, name)
		} else {
			log.Error().Msgf("Invalid value %q of annotation pipy.ingress.kubernetes.io/tls-trusted-ca-secret of Ingress %s/%s: %s", trustedCASecret, ing.Namespace, ing.Name, err)
		}
	}

	// Backend Protocol
	backendProtocol := strings.ToUpper(ing.Annotations[constants.PipyIngressAnnotationBackendProtocol])
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

func (t *IngressChangeTracker) getTLSSecretName(rule *networkingv1.IngressRule, ing *networkingv1.Ingress) string {
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

		cert := t.fetchSSLCert(ing, ing.Namespace, tls.SecretName)
		if cert == nil {
			continue
		}

		if cert.Cert == "" || cert.Key == "" {
			log.Warn().Msgf("Empty Certificate/PrivateKey from secret %s/%s", ing.Namespace, tls.SecretName)
			continue
		}

		x509Cert, err := utils.ConvertPEMCertToX509([]byte(cert.Cert))
		if err != nil {
			log.Warn().Msgf("Failed to convert PEM cert to X509: %s", err)
			continue
		}

		if err := x509Cert.VerifyHostname(host); err != nil {
			log.Warn().Msgf("Failed validating SSL certificate %s/%s for host %q: %v", ing.Namespace, tls.SecretName, host, err)
			continue
		}

		log.Info().Msgf("Found SSL certificate matching host %q: %s/%s", host, ing.Namespace, tls.SecretName)
		return tls.SecretName
	}

	return ""
}

func (t *IngressChangeTracker) fetchSSLCert(ing *networkingv1.Ingress, ns, name string) *repocfg.CertificateSpec {
	if ns == "" {
		log.Warn().Msgf("namespace is empty, will use Ingress's namespace")
		ns = ing.Namespace
	}

	if name == "" {
		log.Error().Msgf("Secret name is empty of Ingress %s/%s", ing.Namespace, ing.Name)
		return nil
	}

	log.Info().Msgf("Fetching secret %s/%s ...", ns, name)
	//secret, err := t.informers.GetListers().Secret.Secrets(ns).Get(name)
	secret := &corev1.Secret{}
	err := t.client.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: name}, secret)
	if err != nil {
		log.Error().Msgf("Failed to get secret %s/%s of Ingress %s/%s: %s", ns, name, ing.Namespace, ing.Name, err)
		return nil
	}

	return &repocfg.CertificateSpec{
		Cert: string(secret.Data[constants.TLSCertName]),
		Key:  string(secret.Data[constants.TLSPrivateKeyName]),
		CA:   string(secret.Data[constants.RootCACertName]),
	}
}
