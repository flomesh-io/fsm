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
	"sort"
	"strings"
	"time"

	"github.com/flomesh-io/fsm/pkg/version"

	k8scache "k8s.io/client-go/tools/cache"

	"github.com/flomesh-io/fsm/pkg/k8s/informers"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"

	"github.com/ghodss/yaml"
	"github.com/go-resty/resty/v2"
	"github.com/sethvargo/go-retry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
	"github.com/flomesh-io/fsm/pkg/flb"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// reconciler reconciles a Service object
type serviceReconciler struct {
	recorder   record.EventRecorder
	fctx       *fctx.ControllerContext
	settingMgr *SettingManager
	cache      map[types.NamespacedName]*corev1.Service
}

func (r *serviceReconciler) NeedLeaderElection() bool {
	return true
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

// NewServiceReconciler returns a new reconciler for Service
func NewServiceReconciler(ctx *fctx.ControllerContext, settingManager *SettingManager) controllers.Reconciler {
	log.Info().Msgf("Creating FLB service reconciler ...")

	recon := &serviceReconciler{
		recorder:   ctx.Manager.GetEventRecorderFor("FLB"),
		fctx:       ctx,
		settingMgr: settingManager,
		cache:      make(map[types.NamespacedName]*corev1.Service),
	}

	ctx.InformerCollection.AddEventHandler(informers.InformerKeyService, k8scache.ResourceEventHandlerFuncs{
		AddFunc:    recon.onSvcAdd,
		UpdateFunc: recon.onSvcUpdate,
		DeleteFunc: recon.onSvcDelete,
	})

	return recon
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
func (r *serviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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
		if result, err := r.settingMgr.CheckSetting(svc); err != nil {
			return result, err
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

		return r.createOrUpdateFLBEntry(ctx, svc)
	}

	return ctrl.Result{}, nil
}

func (r *serviceReconciler) deleteEntryFromFLB(ctx context.Context, svc *corev1.Service) (ctrl.Result, error) {
	//if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
	log.Debug().Msgf("Service %s/%s is being deleted from FLB ...", svc.Namespace, svc.Name)

	setting := r.settingMgr.GetSetting(svc.Namespace)
	result := make(map[string][]string)
	for _, port := range svc.Spec.Ports {
		if !isSupportedProtocol(port.Protocol) {
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
	//}

	return ctrl.Result{}, nil
}

func (r *serviceReconciler) createOrUpdateFLBEntry(ctx context.Context, svc *corev1.Service) (ctrl.Result, error) {
	log.Debug().Msgf("Service %s/%s is being created/updated in FLB ...", svc.Namespace, svc.Name)

	mc := r.fctx.Configurator

	endpoints, err := r.getUpstreams(ctx, svc, mc)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Debug().Msgf("Upstreams of Service %s/%s: %s", svc.Namespace, svc.Name, endpoints)

	params := r.getFLBParameters(svc)

	oldHash := getServiceHash(svc)
	hash := r.computeServiceHash(svc, endpoints, params)
	log.Debug().Msgf("Hash of Service %s/%s: old -> %s, new -> %s", svc.Namespace, svc.Name, oldHash, hash)
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

		return r.updateServiceHash(ctx, svc, hash)
	}

	return ctrl.Result{}, nil
}

func (r *serviceReconciler) updateServiceHash(ctx context.Context, svc *corev1.Service, hash string) (ctrl.Result, error) {
	if len(svc.Annotations) == 0 {
		svc.Annotations = make(map[string]string)
	}

	svc.Annotations[constants.FLBHashAnnotation] = hash

	return ctrl.Result{}, r.fctx.Update(ctx, svc)
}

func (r *serviceReconciler) removeServiceHash(ctx context.Context, svc *corev1.Service) error {
	if len(svc.Annotations) == 0 {
		return nil
	}

	delete(svc.Annotations, constants.FLBHashAnnotation)

	return r.fctx.Update(ctx, svc)
}

func (r *serviceReconciler) computeServiceHash(_ *corev1.Service, endpoints map[string][]string, params map[string]string) string {
	return fmt.Sprintf("%s-%s", utils.SimpleHash(endpoints), utils.SimpleHash(params))
}

func (r *serviceReconciler) getUpstreams(ctx context.Context, svc *corev1.Service, mc configurator.Configurator) (map[string][]string, error) {
	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return nil, nil
	}

	switch mc.GetFLBUpstreamMode() {
	case configv1alpha3.FLBUpstreamModeNodePort:
		return r.getNodePorts(ctx, svc, mc)
	case configv1alpha3.FLBUpstreamModeEndpoint:
		return r.getEndpoints(ctx, svc, mc)
	default:
		return nil, fmt.Errorf("invalid upstream mode %q", mc.GetFLBUpstreamMode())
	}
}

func (r *serviceReconciler) getNodePorts(ctx context.Context, svc *corev1.Service, _ configurator.Configurator) (map[string][]string, error) {
	pods := &corev1.PodList{}
	if err := r.fctx.List(
		ctx,
		pods,
		client.InNamespace(svc.Namespace),
		client.MatchingLabelsSelector{
			Selector: labels.SelectorFromSet(svc.Spec.Selector),
		},
	); err != nil {
		return nil, err
	}

	extIPs := sets.New[string]()
	intIPs := sets.New[string]()

	for _, pod := range pods.Items {
		if pod.Spec.NodeName == "" || pod.Status.PodIP == "" {
			continue
		}

		if !utils.IsPodStatusConditionTrue(pod.Status.Conditions, corev1.PodReady) {
			continue
		}

		node := &corev1.Node{}
		if err := r.fctx.Get(ctx, client.ObjectKey{Name: pod.Spec.NodeName}, node); err != nil {
			if errors.IsNotFound(err) {
				continue
			}

			return nil, err
		}

		for _, addr := range node.Status.Addresses {
			switch addr.Type {
			case corev1.NodeExternalIP:
				extIPs.Insert(addr.Address)
			case corev1.NodeInternalIP:
				intIPs.Insert(addr.Address)
			default:
				continue
			}
		}
	}

	var nodeIPs []string
	if len(extIPs) > 0 {
		nodeIPs = extIPs.UnsortedList()
	} else {
		nodeIPs = intIPs.UnsortedList()
	}

	if version.IsDualStackEnabled(r.fctx.KubeClient) {
		ips, err := utils.FilterByIPFamily(nodeIPs, svc)
		if err != nil {
			return nil, err
		}

		nodeIPs = ips
	}

	setting := r.settingMgr.GetSetting(svc.Namespace)
	result := make(map[string][]string)

	for _, port := range svc.Spec.Ports {
		if !isSupportedProtocol(port.Protocol) {
			continue
		}

		svcKey := serviceKey(setting, svc, port)
		result[svcKey] = make([]string, 0)

		for _, nodeIP := range nodeIPs {
			if port.NodePort <= 0 {
				continue
			}

			result[svcKey] = append(result[svcKey], net.JoinHostPort(nodeIP, fmt.Sprintf("%d", port.NodePort)))
		}
	}

	return result, nil
}

func (r *serviceReconciler) getEndpoints(ctx context.Context, svc *corev1.Service, _ configurator.Configurator) (map[string][]string, error) {
	ep := &corev1.Endpoints{}
	if err := r.fctx.Get(ctx, client.ObjectKeyFromObject(svc), ep); err != nil {
		return nil, err
	}

	setting := r.settingMgr.GetSetting(svc.Namespace)
	result := make(map[string][]string)

	for _, port := range svc.Spec.Ports {
		if !isSupportedProtocol(port.Protocol) {
			continue
		}

		svcKey := serviceKey(setting, svc, port)
		result[svcKey] = make([]string, 0)

		for _, ss := range ep.Subsets {
			matchedPortNameFound := false

			for i, epPort := range ss.Ports {
				targetPort := int32(0)

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
					ep := net.JoinHostPort(epAddress.IP, fmt.Sprintf("%d", targetPort))
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

func (r *serviceReconciler) getFLBParameters(svc *corev1.Service) map[string]string {
	setting := r.settingMgr.GetSetting(svc.Namespace)
	if len(svc.Annotations) == 0 {
		return map[string]string{
			flbAddressPoolHeaderName: setting.flbDefaultAddressPool,
			flbAlgoHeaderName:        getValidAlgo(setting.flbDefaultAlgo),
		}
	}

	params := map[string]string{
		flbAddressPoolHeaderName:          r.getAddressPool(svc),
		flbDesiredIPHeaderName:            svc.Annotations[constants.FLBDesiredIPAnnotation],
		flbMaxConnectionsHeaderName:       svc.Annotations[constants.FLBMaxConnectionsAnnotation],
		flbReadTimeoutHeaderName:          svc.Annotations[constants.FLBReadTimeoutAnnotation],
		flbWriteTimeoutHeaderName:         svc.Annotations[constants.FLBWriteTimeoutAnnotation],
		flbIdleTimeoutHeaderName:          svc.Annotations[constants.FLBIdleTimeoutAnnotation],
		flbAlgoHeaderName:                 r.getAlgorithm(svc),
		flbTagsHeaderName:                 r.getTags(svc),
		flbXForwardedForEnabledHeaderName: svc.Annotations[constants.FLBXForwardedForEnabledAnnotation],
		flbLimitSizeHeaderName:            svc.Annotations[constants.FLBLimitSizeAnnotation],
		flbLimitSyncRateHeaderName:        svc.Annotations[constants.FLBLimitSyncRateAnnotation],
		flbSessionStickyHeaderName:        svc.Annotations[constants.FLBSessionStickyAnnotation],
	}

	if flb.IsTLSEnabled(svc) {
		params[flbTLSEnabledHeaderName] = svc.Annotations[constants.FLBTLSEnabledAnnotation]
		params[flbTLSSecretModeHeaderName] = svc.Annotations[constants.FLBTLSSecretModeAnnotation]
		params[flbTLSPortHeaderName] = svc.Annotations[constants.FLBTLSPortAnnotation]

		switch flb.GetTLSSecretMode(svc) {
		case flb.TLSSecretModeLocal:
			params[flbTLSSecretHeaderName] = secretKey(setting, svc.Namespace, svc.Annotations[constants.FLBTLSSecretAnnotation])
		case flb.TLSSecretModeRemote:
			params[flbTLSSecretHeaderName] = svc.Annotations[constants.FLBTLSSecretAnnotation]
		}
	}

	return params
}

func (r *serviceReconciler) getAddressPool(svc *corev1.Service) string {
	setting := r.settingMgr.GetSetting(svc.Namespace)
	if len(svc.Annotations) == 0 {
		return setting.flbDefaultAddressPool
	}

	pool, ok := svc.Annotations[constants.FLBAddressPoolAnnotation]
	if !ok || len(pool) == 0 {
		return setting.flbDefaultAddressPool
	}

	return pool
}

func (r *serviceReconciler) getAlgorithm(svc *corev1.Service) string {
	setting := r.settingMgr.GetSetting(svc.Namespace)
	if len(svc.Annotations) == 0 {
		return getValidAlgo(setting.flbDefaultAlgo)
	}

	algo, ok := svc.Annotations[constants.FLBAlgoAnnotation]
	if !ok || len(algo) == 0 {
		return getValidAlgo(setting.flbDefaultAlgo)
	}

	return getValidAlgo(algo)
}

func (r *serviceReconciler) getTags(svc *corev1.Service) string {
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

func (r *serviceReconciler) updateFLB(svc *corev1.Service, params map[string]string, result map[string][]string, del bool) (*BalancerAPIResponse, error) {
	setting := r.settingMgr.GetSetting(svc.Namespace)

	if err := setting.UpdateToken(); err != nil {
		log.Error().Msgf("Login to FLB failed: %s", err)
		defer r.recorder.Eventf(svc, corev1.EventTypeWarning, "LoginFailed", "Login to FLB failed: %s", err)

		return nil, err
	}

	var resp *resty.Response
	var statusCode int
	var err error

	if err = retry.Fibonacci(context.TODO(), 1*time.Second, func(ctx context.Context) error {
		resp, statusCode, err = r.invokeFLBAPI(svc.Namespace, params, result, del)

		if err != nil {
			if statusCode == http.StatusUnauthorized {
				if loginErr := setting.ForceUpdateToken(); loginErr != nil {
					log.Error().Msgf("Login to FLB failed: %s", loginErr)
					defer r.recorder.Eventf(svc, corev1.EventTypeWarning, "LoginFailed", "Login to FLB failed: %s", loginErr)

					return loginErr
				}

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

func (r *serviceReconciler) invokeFLBAPI(namespace string, params map[string]string, result map[string][]string, del bool) (*resty.Response, int, error) {
	setting := r.settingMgr.GetSetting(namespace)
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
		resp, err = request.Post(flb.DeleteServiceAPIPath)
	} else {
		resp, err = request.Post(flb.UpdateServiceAPIPath)
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

func (r *serviceReconciler) updateService(ctx context.Context, svc *corev1.Service, _ configurator.Configurator, lbAddresses []string) error {
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

func (r *serviceReconciler) addFinalizer(ctx context.Context, svc *corev1.Service) error {
	if !r.hasFinalizer(ctx, svc) {
		svc.Finalizers = append(svc.Finalizers, finalizerName)
		return r.fctx.Update(ctx, svc)
	}

	return nil
}

func (r *serviceReconciler) removeFinalizer(ctx context.Context, svc *corev1.Service) error {
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

func (r *serviceReconciler) hasFinalizer(_ context.Context, svc *corev1.Service) bool {
	for _, finalizer := range svc.Finalizers {
		if finalizer == finalizerName {
			return true
		}
	}

	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *serviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	bd := ctrl.NewControllerManagedBy(mgr).
		For(
			&corev1.Service{},
			builder.WithPredicates(predicate.NewPredicateFuncs(r.isInterestedService)),
		).
		Watches(
			&corev1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(r.servicesByNamespace),
			builder.WithPredicates(
				predicate.Or(
					predicate.GenerationChangedPredicate{},
					predicate.AnnotationChangedPredicate{},
				),
			),
		)

	switch r.fctx.Configurator.GetFLBUpstreamMode() {
	case configv1alpha3.FLBUpstreamModeNodePort:
		bd = bd.Watches(
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.podToService),
		)
	case configv1alpha3.FLBUpstreamModeEndpoint:
		bd = bd.Watches(
			&corev1.Endpoints{},
			handler.EnqueueRequestsFromMapFunc(r.endpointsToService),
		)
	}

	return bd.Complete(r)
}

func (r *serviceReconciler) isInterestedService(obj client.Object) bool {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		log.Warn().Msgf("unexpected object type: %T", obj)
		return false
	}

	return flb.IsFLBEnabled(svc, r.fctx.KubeClient)
}

func (r *serviceReconciler) podToService(ctx context.Context, pod client.Object) []reconcile.Request {
	allServices := &corev1.ServiceList{}
	if err := r.fctx.List(
		ctx,
		allServices,
		client.InNamespace(pod.GetNamespace()),
	); err != nil {
		log.Warn().Msgf("failed to list services in ns %s: %s", pod.GetNamespace(), err)
		return nil
	}

	if len(allServices.Items) == 0 {
		return nil
	}

	services := make(map[types.NamespacedName]struct{})
	for _, service := range allServices.Items {
		service := service // fix lint GO-LOOP-REF

		if service.Spec.Selector == nil {
			// services with nil selectors match nothing, not everything.
			continue
		}

		if !flb.IsFLBEnabled(&service, r.fctx.KubeClient) {
			continue
		}

		selector := labels.Set(service.Spec.Selector).AsSelectorPreValidated()
		if selector.Matches(labels.Set(pod.GetLabels())) {
			services[client.ObjectKeyFromObject(&service)] = struct{}{}
		}
	}

	if len(services) == 0 {
		return nil
	}

	requests := make([]reconcile.Request, len(services))
	for svc := range services {
		requests = append(requests, reconcile.Request{NamespacedName: svc})
	}

	return requests
}

func (r *serviceReconciler) endpointsToService(ctx context.Context, ep client.Object) []reconcile.Request {
	svc := &corev1.Service{}
	if err := r.fctx.Get(
		ctx,
		client.ObjectKeyFromObject(ep),
		svc,
	); err != nil {
		log.Warn().Msgf("failed to get service %s/%s: %s", ep.GetNamespace(), ep.GetName(), err)
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

func (r *serviceReconciler) servicesByNamespace(ctx context.Context, ns client.Object) []reconcile.Request {
	services, err := r.fctx.KubeClient.CoreV1().
		Services(ns.GetName()).
		List(ctx, metav1.ListOptions{})

	if err != nil {
		log.Warn().Msgf("failed to list services in ns %s: %s", ns.GetName(), err)
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
