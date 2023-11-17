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

// Package flb implements the controller for Flomesh Load Balancer.
package flb

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/go-resty/resty/v2"
	"github.com/sethvargo/go-retry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
	"github.com/flomesh-io/fsm/pkg/flb"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// FLB API paths
const (
	flbAuthAPIPath          = "/api/auth/local"
	flbUpdateServiceAPIPath = "/api/l-4-lbs/updateservice"
	flbDeleteServiceAPIPath = "/api/l-4-lbs/updateservice/delete"
)

// FLB annotations
const (
	finalizerName        = "servicelb.flomesh.io/flb"
	flbDefaultSettingKey = "flb.flomesh.io/default-setting"
)

// FLB request HTTP headers
const (
	flbAddressPoolHeaderName    = "X-Flb-Address-Pool"
	flbDesiredIPHeaderName      = "X-Flb-Desired-Ip"
	flbMaxConnectionsHeaderName = "X-Flb-Max-Connections"
	flbReadTimeoutHeaderName    = "X-Flb-Read-Timeout"
	flbWriteTimeoutHeaderName   = "X-Flb-Write-Timeout"
	flbIdleTimeoutHeaderName    = "X-Flb-Idle-Timeout"
	flbAlgoHeaderName           = "X-Flb-Algo"
	flbUserHeaderName           = "X-Flb-User"
	flbK8sClusterHeaderName     = "X-Flb-K8s-Cluster"
	flbTagsHeaderName           = "X-Flb-Tags"
)

// reconciler reconciles a Service object
type reconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
	settings map[string]*setting
	cache    map[types.NamespacedName]*corev1.Service
}

func (r *reconciler) NeedLeaderElection() bool {
	return true
}

// setting is the setting for a FLB instance per namespace
type setting struct {
	httpClient            *resty.Client
	flbUser               string
	flbPassword           string
	k8sCluster            string
	flbDefaultAddressPool string
	flbDefaultAlgo        string
	token                 string
	hash                  string
}

// AuthRequest is the request body for FLB authentication
type AuthRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// AuthResponse is the response body for FLB authentication
type AuthResponse struct {
	Token string `json:"jwt"`
}

// BalancerAPIResponse is the response body for FLB API
type BalancerAPIResponse struct {
	LBIPs []string `json:"LBIPs"`
}

// serviceTag is the tags for a service port
type serviceTag struct {
	Port int32             `json:"port"`
	Tags map[string]string `json:"tags"`
}

var (
	log = logger.New("flb-service-controller")
)

// NewReconciler returns a new reconciler for Service
func NewReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	log.Info().Msgf("Creating FLB service reconciler ...")

	mc := ctx.Config
	if !mc.IsFLBEnabled() {
		panic("FLB is not enabled")
	}

	if mc.GetFLBSecretName() == "" {
		panic("FLB Secret Name is empty, it's required.")
	}

	settings := make(map[string]*setting)

	// get default settings
	defaultSetting, err := getDefaultSetting(ctx.KubeClient, mc)
	if err != nil {
		panic(err)
	}
	settings[flbDefaultSettingKey] = defaultSetting

	secrets, err := ctx.KubeClient.CoreV1().
		Secrets(corev1.NamespaceAll).
		List(context.TODO(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", mc.GetFLBSecretName()),
			LabelSelector: labels.SelectorFromSet(
				map[string]string{constants.FLBSecretLabel: "true"},
			).String(),
		})

	if err != nil {
		panic(err)
	}

	for _, secret := range secrets.Items {
		secret := secret // fix lint GO-LOOP-REF
		if mc.IsFLBStrictModeEnabled() {
			settings[secret.Namespace] = newSetting(&secret)
		} else {
			settings[secret.Namespace] = newOverrideSetting(&secret, defaultSetting)
		}
	}

	return &reconciler{
		recorder: ctx.Manager.GetEventRecorderFor("FLB"),
		fctx:     ctx,
		settings: settings,
		cache:    make(map[types.NamespacedName]*corev1.Service),
	}
}

func getDefaultSetting(api kubernetes.Interface, mc configurator.Configurator) (*setting, error) {
	secret, err := api.CoreV1().
		Secrets(mc.GetFSMNamespace()).
		Get(context.TODO(), mc.GetFLBSecretName(), metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	if !secretHasRequiredLabel(secret) {
		return nil, fmt.Errorf("secret %s/%s doesn't have required label %s=true", mc.GetFSMNamespace(), mc.GetFLBSecretName(), constants.FLBSecretLabel)
	}

	log.Debug().Msgf("Found Secret %s/%s", mc.GetFSMNamespace(), mc.GetFLBSecretName())

	log.Debug().Msgf("FLB base URL = %q", string(secret.Data[constants.FLBSecretKeyBaseURL]))
	log.Debug().Msgf("FLB default Address Pool = %q", string(secret.Data[constants.FLBSecretKeyDefaultAddressPool]))

	return newSetting(secret), nil
}

func newSetting(secret *corev1.Secret) *setting {
	return &setting{
		httpClient:            newHTTPClient(string(secret.Data[constants.FLBSecretKeyBaseURL])),
		flbUser:               string(secret.Data[constants.FLBSecretKeyUsername]),
		flbPassword:           string(secret.Data[constants.FLBSecretKeyPassword]),
		k8sCluster:            string(secret.Data[constants.FLBSecretKeyK8sCluster]),
		flbDefaultAddressPool: string(secret.Data[constants.FLBSecretKeyDefaultAddressPool]),
		flbDefaultAlgo:        string(secret.Data[constants.FLBSecretKeyDefaultAlgo]),
		hash:                  fmt.Sprintf("%d", utils.GetSecretDataHash(secret)),
		token:                 "",
	}
}

func newOverrideSetting(secret *corev1.Secret, defaultSetting *setting) *setting {
	s := &setting{
		hash:  fmt.Sprintf("%d-%s", utils.GetSecretDataHash(secret), defaultSetting.hash),
		token: "",
	}

	if len(secret.Data[constants.FLBSecretKeyBaseURL]) == 0 {
		s.httpClient = defaultSetting.httpClient
	} else {
		s.httpClient = newHTTPClient(string(secret.Data[constants.FLBSecretKeyBaseURL]))
	}

	if len(secret.Data[constants.FLBSecretKeyUsername]) == 0 {
		s.flbUser = defaultSetting.flbUser
	} else {
		s.flbUser = string(secret.Data[constants.FLBSecretKeyUsername])
	}

	if len(secret.Data[constants.FLBSecretKeyPassword]) == 0 {
		s.flbPassword = defaultSetting.flbPassword
	} else {
		s.flbPassword = string(secret.Data[constants.FLBSecretKeyPassword])
	}

	if len(secret.Data[constants.FLBSecretKeyK8sCluster]) == 0 {
		s.k8sCluster = defaultSetting.k8sCluster
	} else {
		s.k8sCluster = string(secret.Data[constants.FLBSecretKeyK8sCluster])
	}

	if len(secret.Data[constants.FLBSecretKeyDefaultAddressPool]) == 0 {
		s.flbDefaultAddressPool = defaultSetting.flbDefaultAddressPool
	} else {
		s.flbDefaultAddressPool = string(secret.Data[constants.FLBSecretKeyDefaultAddressPool])
	}

	if len(secret.Data[constants.FLBSecretKeyDefaultAlgo]) == 0 {
		s.flbDefaultAlgo = defaultSetting.flbDefaultAlgo
	} else {
		s.flbDefaultAlgo = string(secret.Data[constants.FLBSecretKeyDefaultAlgo])
	}

	return s
}

func newHTTPClient(baseURL string) *resty.Client {
	return resty.New().
		SetTransport(&http.Transport{
			DisableKeepAlives:  false,
			MaxIdleConns:       10,
			IdleConnTimeout:    60 * time.Second,
			DisableCompression: false,
		}).
		SetScheme("http").
		SetBaseURL(baseURL).
		SetTimeout(5 * time.Second).
		SetDebug(true).
		EnableTrace()
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Service object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the Service instance
	svc := &corev1.Service{}
	if err := r.fctx.Get(
		ctx,
		req.NamespacedName,
		svc,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info().Msgf("Service %s/%s resource not found. Ignoring since object must be deleted", req.Namespace, req.Name)

			// get svc from cache as it's not found, we don't have enough info to pop out
			svc, ok := r.cache[req.NamespacedName]
			if !ok {
				log.Warn().Msgf("Service %s not found in cache", req.NamespacedName)
				return ctrl.Result{}, nil
			}

			if flb.IsFLBEnabled(svc, r.fctx.KubeClient) {
				result, err := r.deleteEntryFromFLB(ctx, svc)
				if err != nil {
					return result, err
				}

				delete(r.cache, req.NamespacedName)
				return ctrl.Result{}, nil
			}
		}
		// Error reading the object - requeue the request.
		log.Error().Msgf("Failed to get Service, %#v", err)
		return ctrl.Result{}, err
	}

	if flb.IsFLBEnabled(svc, r.fctx.KubeClient) {
		log.Debug().Msgf("Type of service %s/%s is LoadBalancer", req.Namespace, req.Name)

		//oldSvc, found := r.cache[req.NamespacedName]
		//if found && oldSvc.ResourceVersion == svc.ResourceVersion {
		//	log.Info().Msgf("Service %s/%s hasn't changed or not processed yet, ResourceRevision=%s, skipping ...", req.Namespace, req.Name, svc.ResourceVersion)
		//	return ctrl.Result{}, nil
		//}

		r.cache[req.NamespacedName] = svc.DeepCopy()
		mc := r.fctx.Config

		secrets, err := r.fctx.KubeClient.CoreV1().
			Secrets(svc.Namespace).
			List(context.TODO(), metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", mc.GetFLBSecretName()),
				LabelSelector: labels.SelectorFromSet(
					map[string]string{constants.FLBSecretLabel: "true"},
				).String(),
			})

		if err != nil {
			defer r.recorder.Eventf(svc, corev1.EventTypeWarning, "GetSecretFailed", "Failed to get FLB secret %s/%s", svc.Namespace, mc.GetFLBSecretName())
			return ctrl.Result{}, err
		}

		switch len(secrets.Items) {
		case 0:
			if mc.IsFLBStrictModeEnabled() {
				defer r.recorder.Eventf(svc, corev1.EventTypeWarning, "GetSecretFailed", "In StrictMode, FLB secret %s/%s must exist", svc.Namespace, mc.GetFLBSecretName())
				return ctrl.Result{}, err
			}

			if r.settings[svc.Namespace] == nil {
				defer r.recorder.Eventf(svc, corev1.EventTypeNormal, "UseDefaultSecret", "FLB Secret %s/%s doesn't exist, using default ...", svc.Namespace, mc.GetFLBSecretName())
				r.settings[svc.Namespace] = r.settings[flbDefaultSettingKey]
			}
		case 1:
			secret := &secrets.Items[0]

			if r.settings[svc.Namespace] == nil {
				if mc.IsFLBStrictModeEnabled() {
					r.settings[svc.Namespace] = newSetting(secret)
				} else {
					r.settings[svc.Namespace] = newOverrideSetting(secret, r.settings[flbDefaultSettingKey])
				}
			} else {
				setting := r.settings[svc.Namespace]
				if isSettingChanged(secret, setting, r.settings[flbDefaultSettingKey], mc) {
					if svc.Namespace == mc.GetFSMNamespace() {
						r.settings[flbDefaultSettingKey] = newSetting(secret)
					}

					if mc.IsFLBStrictModeEnabled() {
						r.settings[svc.Namespace] = newSetting(secret)
					} else {
						r.settings[svc.Namespace] = newOverrideSetting(secret, r.settings[flbDefaultSettingKey])
					}
				}
			}
		}

		if svc.DeletionTimestamp != nil {
			result, err := r.deleteEntryFromFLB(ctx, svc)
			if err != nil {
				return result, err
			}

			delete(r.cache, req.NamespacedName)
			return ctrl.Result{}, nil
		}

		log.Debug().Msgf("Annotations of service %s/%s is %v", svc.Namespace, svc.Name, svc.Annotations)
		if newAnnotations := r.computeServiceAnnotations(svc); newAnnotations != nil {
			svc.Annotations = newAnnotations
			if err := r.fctx.Update(ctx, svc); err != nil {
				log.Error().Msgf("Failed update annotations of service %s/%s: %s", svc.Namespace, svc.Name, err)
				return ctrl.Result{}, err
			}

			log.Debug().Msgf("After updating, annotations of service %s/%s is %v", svc.Namespace, svc.Name, svc.Annotations)
		}

		return r.createOrUpdateFLBEntry(ctx, svc)
	}

	return ctrl.Result{}, nil
}

func (r *reconciler) computeServiceAnnotations(svc *corev1.Service) map[string]string {
	setting := r.settings[svc.Namespace]
	log.Debug().Msgf("Setting for Namespace %q: %v", svc.Namespace, setting)

	svcCopy := svc.DeepCopy()
	if svcCopy.Annotations == nil {
		svcCopy.Annotations = make(map[string]string)
	}

	for key, value := range map[string]string{
		constants.FLBAddressPoolAnnotation: setting.flbDefaultAddressPool,
		constants.FLBAlgoAnnotation:        getValidAlgo(setting.flbDefaultAlgo),
	} {
		v, ok := svcCopy.Annotations[key]
		if !ok || v == "" {
			svcCopy.Annotations[key] = value
		}
	}

	if !reflect.DeepEqual(svc.GetAnnotations(), svcCopy.GetAnnotations()) {
		return svcCopy.Annotations
	}

	return nil
}

func isSettingChanged(secret *corev1.Secret, setting, defaultSetting *setting, mc configurator.Configurator) bool {
	if mc.IsFLBStrictModeEnabled() {
		hash := fmt.Sprintf("%d", utils.GetSecretDataHash(secret))
		if hash != setting.hash {
			return true
		}
	} else {
		hash := fmt.Sprintf("%d-%s", utils.GetSecretDataHash(secret), defaultSetting.hash)
		if hash != setting.hash {
			return true
		}
	}

	return false
}

func secretHasRequiredLabel(secret *corev1.Secret) bool {
	if len(secret.Labels) == 0 {
		return false
	}

	value, ok := secret.Labels[constants.FLBSecretLabel]
	if !ok {
		return false
	}

	return value == "true"
}

func (r *reconciler) deleteEntryFromFLB(ctx context.Context, svc *corev1.Service) (ctrl.Result, error) {
	if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
		log.Debug().Msgf("Service %s/%s is being deleted from FLB ...", svc.Namespace, svc.Name)

		setting := r.settings[svc.Namespace]
		result := make(map[string][]string)
		for _, port := range svc.Spec.Ports {
			if !isSupportedProtocol(port) {
				continue
			}

			svcKey := serviceKey(setting, svc, port)
			result[svcKey] = make([]string, 0)
		}

		params := r.getFLBParameters(svc)
		if _, err := r.updateFLB(svc, params, result, true); err != nil {
			return ctrl.Result{}, err
		}

		if svc.DeletionTimestamp != nil {
			return ctrl.Result{}, r.removeFinalizer(ctx, svc)
		}
	}

	return ctrl.Result{}, nil
}

func (r *reconciler) createOrUpdateFLBEntry(ctx context.Context, svc *corev1.Service) (ctrl.Result, error) {
	log.Debug().Msgf("Service %s/%s is being created/updated in FLB ...", svc.Namespace, svc.Name)

	mc := r.fctx.Config

	endpoints, err := r.getEndpoints(ctx, svc, mc)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Debug().Msgf("Endpoints of Service %s/%s: %s", svc.Namespace, svc.Name, endpoints)

	params := r.getFLBParameters(svc)

	oldHash := getServiceHash(svc)
	hash := fmt.Sprintf("%s-%s", utils.SimpleHash(endpoints), utils.SimpleHash(params))
	if oldHash != hash {
		resp, err := r.updateFLB(svc, params, endpoints, false)
		if err != nil {
			return ctrl.Result{}, err
		}

		if len(resp.LBIPs) == 0 {
			// it should always assign a VIP for the service, not matter it has endpoints or not
			defer r.recorder.Eventf(svc, corev1.EventTypeWarning, "IPNotAssigned", "FLB hasn't assigned any external IP yet")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("FLB hasn't assigned any external IP for service %s/%s", svc.Namespace, svc.Name)
		}

		log.Debug().Msgf("External IPs assigned by FLB: %#v", resp)

		if err := r.updateService(ctx, svc, mc, resp.LBIPs); err != nil {
			return ctrl.Result{}, err
		}

		svc.Annotations[constants.FLBServiceHashAnnotation] = hash
		if err := r.fctx.Update(ctx, svc); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func getServiceHash(svc *corev1.Service) string {
	if len(svc.Annotations) == 0 {
		return ""
	}
	return svc.Annotations[constants.FLBServiceHashAnnotation]
}

func (r *reconciler) getEndpoints(ctx context.Context, svc *corev1.Service, _ configurator.Configurator) (map[string][]string, error) {
	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return nil, nil
	}

	ep := &corev1.Endpoints{}
	if err := r.fctx.Get(ctx, client.ObjectKeyFromObject(svc), ep); err != nil {
		return nil, err
	}

	setting := r.settings[svc.Namespace]
	result := make(map[string][]string)

	for _, port := range svc.Spec.Ports {
		if !isSupportedProtocol(port) {
			continue
		}

		svcKey := serviceKey(setting, svc, port)
		result[svcKey] = make([]string, 0)

		for _, ss := range ep.Subsets {
			matchedPortNameFound := false

			for i, epPort := range ss.Ports {
				if epPort.Protocol != corev1.ProtocolTCP {
					continue
				}

				var targetPort int32

				if port.Name == "" {
					// port.Name is optional if there is only one port
					targetPort = epPort.Port
					matchedPortNameFound = true
				} else if port.Name == epPort.Name {
					targetPort = epPort.Port
					matchedPortNameFound = true
				}

				if i == len(ss.Ports)-1 && !matchedPortNameFound && port.TargetPort.Type == intstr.Int {
					targetPort = port.TargetPort.IntVal
				}

				if targetPort <= 0 {
					continue
				}

				for _, epAddress := range ss.Addresses {
					ep := net.JoinHostPort(epAddress.IP, strconv.Itoa(int(targetPort)))
					result[svcKey] = append(result[svcKey], ep)
				}
			}
		}
	}

	// for each service key, sort the endpoints to make sure the order is consistent
	for svcKey, eps := range result {
		sort.Strings(eps)
		result[svcKey] = eps
	}

	return result, nil
}

func isSupportedProtocol(port corev1.ServicePort) bool {
	switch port.Protocol {
	case corev1.ProtocolTCP, corev1.ProtocolUDP:
		return true
	default:
		return false
	}
}

func (r *reconciler) getFLBParameters(svc *corev1.Service) map[string]string {
	if svc.Annotations == nil {
		return map[string]string{}
	}

	return map[string]string{
		flbAddressPoolHeaderName:    svc.Annotations[constants.FLBAddressPoolAnnotation],
		flbDesiredIPHeaderName:      svc.Annotations[constants.FLBDesiredIPAnnotation],
		flbMaxConnectionsHeaderName: svc.Annotations[constants.FLBMaxConnectionsAnnotation],
		flbReadTimeoutHeaderName:    svc.Annotations[constants.FLBReadTimeoutAnnotation],
		flbWriteTimeoutHeaderName:   svc.Annotations[constants.FLBWriteTimeoutAnnotation],
		flbIdleTimeoutHeaderName:    svc.Annotations[constants.FLBIdleTimeoutAnnotation],
		flbAlgoHeaderName:           getValidAlgo(svc.Annotations[constants.FLBAlgoAnnotation]),
		flbTagsHeaderName:           r.getTags(svc),
	}
}

func (r *reconciler) getTags(svc *corev1.Service) string {
	rawTags, ok := svc.Annotations[constants.FLBTagsAnnotation]

	if !ok || len(rawTags) == 0 {
		return ""
	}

	tags := make([]serviceTag, 0)
	if err := yaml.Unmarshal([]byte(rawTags), &tags); err != nil {
		log.Error().Msgf("Failed to unmarshal tags: %s, it' not in a valid format", err)
		defer r.recorder.Eventf(svc, corev1.EventTypeWarning, "InvalidTagFormat", "Format of annotation %s is not valid", constants.FLBTagsAnnotation)
		return ""
	}
	log.Debug().Msgf("Unmarshalled tags of service %s/%s: %v", svc.Namespace, svc.Name, tags)

	svcPorts := make(map[int32]bool)
	for _, port := range svc.Spec.Ports {
		svcPorts[port.Port] = true
	}
	log.Debug().Msgf("Ports of service %s/%s: %v", svc.Namespace, svc.Name, svcPorts)

	resultTags := make([]serviceTag, 0)
	for _, tag := range tags {
		if _, ok := svcPorts[tag.Port]; !ok {
			continue
		}
		resultTags = append(resultTags, tag)
	}
	log.Debug().Msgf("Valid tags for service %s/%s: %v", svc.Namespace, svc.Name, resultTags)

	if len(resultTags) == 0 {
		return ""
	}

	resultTagsBytes, err := json.Marshal(resultTags)
	if err != nil {
		log.Error().Msgf("Failed to marshal tags: %s", err)
		defer r.recorder.Eventf(svc, corev1.EventTypeWarning, "MarshalJson", "Failed marshal tags to JSON: %s", err)
		return ""
	}

	tagsJson := string(resultTagsBytes)
	log.Debug().Msgf("tagsJson: %s", tagsJson)

	return tagsJson
}

func (r *reconciler) updateFLB(svc *corev1.Service, params map[string]string, result map[string][]string, del bool) (*BalancerAPIResponse, error) {
	if r.settings[svc.Namespace].token == "" {
		token, err := r.loginFLB(svc.Namespace)
		if err != nil {
			log.Error().Msgf("Login to FLB failed: %s", err)
			defer r.recorder.Eventf(svc, corev1.EventTypeWarning, "LoginFailed", "Login to FLB failed: %s", err)

			return nil, err
		}

		r.settings[svc.Namespace].token = token
	}

	var resp *resty.Response
	var statusCode int
	var err error

	if err = retry.Fibonacci(context.TODO(), 1*time.Second, func(ctx context.Context) error {
		resp, statusCode, err = r.invokeFLBAPI(svc.Namespace, params, result, del)

		if err != nil {
			if statusCode == http.StatusUnauthorized {
				token, loginErr := r.loginFLB(svc.Namespace)
				if loginErr != nil {
					log.Error().Msgf("Login to FLB failed: %s", loginErr)
					defer r.recorder.Eventf(svc, corev1.EventTypeWarning, "LoginFailed", "Login to FLB failed: %s", loginErr)

					return loginErr
				}

				r.settings[svc.Namespace].token = token

				return retry.RetryableError(err)
			}

			defer r.recorder.Eventf(svc, corev1.EventTypeWarning, "InvokeFLBApiError", "Failed to invoke FLB API: %s", err)
			return err
		}

		return nil
	}); err != nil {
		log.Error().Msgf("failed to update FLB: %s", err)
		defer r.recorder.Eventf(svc, corev1.EventTypeWarning, "UpdateFLBFailed", "Failed to update FLB: %s", err)

		return nil, err
	}

	return resp.Result().(*BalancerAPIResponse), nil
}

func (r *reconciler) invokeFLBAPI(namespace string, params map[string]string, result map[string][]string, del bool) (*resty.Response, int, error) {
	setting := r.settings[namespace]
	request := setting.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetHeader(flbUserHeaderName, setting.flbUser).
		SetHeader(flbK8sClusterHeaderName, setting.k8sCluster).
		SetAuthToken(setting.token).
		SetBody(result).
		SetResult(&BalancerAPIResponse{})

	for h, v := range params {
		if v != "" {
			request.SetHeader(h, v)
		}
	}

	var resp *resty.Response
	var err error
	if del {
		resp, err = request.Post(flbDeleteServiceAPIPath)
	} else {
		resp, err = request.Post(flbUpdateServiceAPIPath)
	}

	if err != nil {
		log.Error().Msgf("error happened while trying to update FLB, %s", err.Error())
		return nil, -1, err
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		return nil, http.StatusUnauthorized, fmt.Errorf("invalid token")
	}

	if resp.StatusCode() != http.StatusOK {
		log.Error().Msgf("FLB server responsed with StatusCode: %d", resp.StatusCode())
		return nil, resp.StatusCode(), fmt.Errorf("%d: %s", resp.StatusCode(), string(resp.Body()))
	}

	return resp, http.StatusOK, nil
}

func (r *reconciler) loginFLB(namespace string) (string, error) {
	setting := r.settings[namespace]
	resp, err := setting.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(AuthRequest{Identifier: setting.flbUser, Password: setting.flbPassword}).
		SetResult(&AuthResponse{}).
		Post(flbAuthAPIPath)

	if err != nil {
		log.Error().Msgf("error happened while trying to login FLB, %s", err.Error())
		return "", err
	}

	if resp.StatusCode() != http.StatusOK {
		log.Error().Msgf("FLB server responsed with StatusCode: %d", resp.StatusCode())
		return "", fmt.Errorf("StatusCode: %d", resp.StatusCode())
	}

	return resp.Result().(*AuthResponse).Token, nil
}

func (r *reconciler) updateService(ctx context.Context, svc *corev1.Service, _ configurator.Configurator, lbAddresses []string) error {
	if svc.DeletionTimestamp != nil || svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return r.removeFinalizer(ctx, svc)
	}

	existingIPs := serviceIPs(svc)
	expectedIPs := lbIPs(lbAddresses)

	sort.Strings(expectedIPs)
	sort.Strings(existingIPs)

	if utils.StringsEqual(expectedIPs, existingIPs) {
		return nil
	}

	svc = svc.DeepCopy()
	if err := r.addFinalizer(ctx, svc); err != nil {
		return err
	}

	svc.Status.LoadBalancer.Ingress = nil
	for _, ip := range expectedIPs {
		svc.Status.LoadBalancer.Ingress = append(svc.Status.LoadBalancer.Ingress, corev1.LoadBalancerIngress{
			IP: ip,
		})
	}

	defer r.recorder.Eventf(svc, corev1.EventTypeNormal, "UpdatedIngressIP", "LoadBalancer Ingress IP addresses updated: %s", strings.Join(expectedIPs, ", "))

	return r.fctx.Status().Update(ctx, svc)
}

func lbIPs(addresses []string) []string {
	if len(addresses) == 0 {
		return nil
	}

	ips := make([]string, 0)
	for _, addr := range addresses {
		if strings.Contains(addr, ":") {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil
			}
			ips = append(ips, host)
		} else {
			ips = append(ips, addr)
		}
	}

	return ips
}

// serviceIPs returns the list of ingress IP addresses from the Service
func serviceIPs(svc *corev1.Service) []string {
	var ips []string

	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			ips = append(ips, ingress.IP)
		}
	}

	return ips
}

func serviceKey(setting *setting, svc *corev1.Service, port corev1.ServicePort) string {
	return fmt.Sprintf("%s/%s/%s:%d#%s", setting.k8sCluster, svc.Namespace, svc.Name, port.Port, strings.ToUpper(string(port.Protocol)))
}

func (r *reconciler) addFinalizer(ctx context.Context, svc *corev1.Service) error {
	if !r.hasFinalizer(ctx, svc) {
		svc.Finalizers = append(svc.Finalizers, finalizerName)
		return r.fctx.Update(ctx, svc)
	}

	return nil
}

func (r *reconciler) removeFinalizer(ctx context.Context, svc *corev1.Service) error {
	if !r.hasFinalizer(ctx, svc) {
		return nil
	}

	for k, v := range svc.Finalizers {
		if v != finalizerName {
			continue
		}
		svc.Finalizers = append(svc.Finalizers[:k], svc.Finalizers[k+1:]...)
	}

	return r.fctx.Update(ctx, svc)
}

func (r *reconciler) hasFinalizer(_ context.Context, svc *corev1.Service) bool {
	for _, finalizer := range svc.Finalizers {
		if finalizer == finalizerName {
			return true
		}
	}

	return false
}

func getValidAlgo(value string) string {
	switch value {
	case "rr", "lc", "ch":
		return value
	default:
		log.Warn().Msgf("Invalid ALGO value %q, will use 'rr' as default", value)
		return "rr"
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&corev1.Service{},
			builder.WithPredicates(predicate.NewPredicateFuncs(r.isInterestedService)),
		).
		Watches(
			&source.Kind{Type: &corev1.Endpoints{}},
			handler.EnqueueRequestsFromMapFunc(r.endpointsToService),
		).
		Watches(
			&source.Kind{Type: &corev1.Namespace{}},
			handler.EnqueueRequestsFromMapFunc(r.servicesByNamespace),
			builder.WithPredicates(
				predicate.Or(
					predicate.GenerationChangedPredicate{},
					predicate.AnnotationChangedPredicate{},
				),
			),
		).
		Complete(r)
}

func (r *reconciler) isInterestedService(obj client.Object) bool {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		log.Debug().Msgf("unexpected object type: %T", obj)
		return false
	}

	return flb.IsFLBEnabled(svc, r.fctx.KubeClient)
}

func (r *reconciler) endpointsToService(ep client.Object) []reconcile.Request {
	svc := &corev1.Service{}
	if err := r.fctx.Get(
		context.TODO(),
		client.ObjectKeyFromObject(ep),
		svc,
	); err != nil {
		log.Error().Msgf("failed to get service %s/%s: %s", ep.GetNamespace(), ep.GetName(), err)
		return nil
	}

	// ONLY if it's FLB interested service
	if flb.IsFLBEnabled(svc, r.fctx.KubeClient) {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: svc.GetNamespace(),
					Name:      svc.GetName(),
				},
			},
		}
	}

	return nil
}

func (r *reconciler) servicesByNamespace(ns client.Object) []reconcile.Request {
	services, err := r.fctx.KubeClient.CoreV1().
		Services(ns.GetName()).
		List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		log.Error().Msgf("failed to list services in ns %s: %s", ns.GetName(), err)
		return nil
	}

	requests := make([]reconcile.Request, 0)

	for _, svc := range services.Items {
		svc := svc // fix lint GO-LOOP-REF
		if flb.IsFLBEnabled(&svc, r.fctx.KubeClient) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: svc.GetNamespace(),
					Name:      svc.GetName(),
				},
			})
		}
	}

	return requests
}
