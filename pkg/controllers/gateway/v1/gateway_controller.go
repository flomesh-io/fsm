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

package v1

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	ghodssyaml "github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"

	"github.com/flomesh-io/fsm/pkg/gateway/status"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/flomesh-io/fsm/pkg/version"

	"sigs.k8s.io/yaml"

	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"

	"github.com/flomesh-io/fsm/pkg/helm"
	"github.com/flomesh-io/fsm/pkg/utils"
)

var (
	//go:embed chart.tgz
	chartSource []byte

	// namespace <-> active gateway
	activeGateways map[string]*gwv1.Gateway
)

//type gatewayValues struct {
//	Gateway   *gwv1.Gateway    `json:"gwy,omitempty"`
//	Listeners []gwpkg.Listener `json:"listeners,omitempty"`
//}

type gatewayAcceptedCondition struct {
	gateway   *gwv1.Gateway
	condition metav1.Condition
}

type Listener struct {
	Name     gwv1.SectionName  `json:"name"`
	Port     gwv1.PortNumber   `json:"port"`
	Protocol gwv1.ProtocolType `json:"protocol"`
}

type gatewayReconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *gatewayReconciler) NeedLeaderElection() bool {
	return true
}

func init() {
	activeGateways = make(map[string]*gwv1.Gateway)
}

// NewGatewayReconciler returns a new reconciler for Gateway resources
func NewGatewayReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &gatewayReconciler{
		recorder: ctx.Manager.GetEventRecorderFor("Gateway"),
		fctx:     ctx,
	}
}

// Reconcile reconciles a Gateway resource
func (r *gatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	gateway := &gwv1.Gateway{}
	if err := r.fctx.Get(
		ctx,
		req.NamespacedName,
		gateway,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info().Msgf("Gateway resource not found. Ignoring since object must be deleted")
			r.fctx.GatewayEventHandler.OnDelete(&gwv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: req.Namespace,
					Name:      req.Name,
				}})
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error().Msgf("Failed to get Gateway, %v", err)
		return ctrl.Result{}, err
	}

	if gateway.DeletionTimestamp != nil {
		r.fctx.GatewayEventHandler.OnDelete(gateway)
		return ctrl.Result{}, nil
	}

	log.Debug().Msgf("[GW] 1")
	effectiveGatewayClass, err := gwutils.FindEffectiveGatewayClass(r.fctx.Manager.GetCache())
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Debug().Msgf("[GW] 2")
	if effectiveGatewayClass == nil {
		log.Warn().Msgf("No effective GatewayClass, ignore processing Gateway resource %s.", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	log.Debug().Msgf("[GW] 3")
	result, err := r.updateGatewayStatus(ctx, gateway, effectiveGatewayClass)
	if err != nil {
		return result, err
	}

	log.Debug().Msgf("[GW] 4")
	// 5. update listener status of this gateway no matter it's accepted or not
	result, err = r.updateListenerStatus(ctx, gateway)
	if err != nil {
		return result, err
	}

	log.Debug().Msgf("[GW] 5")
	result, err = r.updateGatewayAddresses(ctx, gateway)
	if err != nil || result.RequeueAfter > 0 || result.Requeue {
		return result, err
	}

	log.Debug().Msgf("[GW] 6")
	r.fctx.GatewayEventHandler.OnAdd(gateway, false)

	log.Debug().Msgf("[GW] 7")
	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) updateGatewayStatus(ctx context.Context, gateway *gwv1.Gateway, effectiveGatewayClass *gwv1.GatewayClass) (ctrl.Result, error) {
	statusChangedGateways, err := r.computeGatewayAcceptedCondition(ctx, gateway, effectiveGatewayClass)
	if err != nil {
		return ctrl.Result{}, err
	}

	// 4. update status
	for _, g := range statusChangedGateways {
		r.fctx.StatusUpdater.Send(status.Update{
			Resource:       &gwv1.Gateway{},
			NamespacedName: client.ObjectKeyFromObject(g.gateway),
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				gw, ok := obj.(*gwv1.Gateway)
				if !ok {
					log.Error().Msgf("Unexpected object type %T", obj)
				}
				gwCopy := gw.DeepCopy()
				metautil.SetStatusCondition(&gwCopy.Status.Conditions, g.condition)

				if gwutils.IsAcceptedGateway(gwCopy) {
					defer r.recorder.Eventf(gwCopy, corev1.EventTypeNormal, "Accepted", "Gateway is accepted")
				} else {
					defer r.recorder.Eventf(gwCopy, corev1.EventTypeWarning, "Rejected", "Gateway in not accepted due to it's not the oldest in namespace %s or its gatewayClassName is incorrect", gwCopy.Namespace)
				}

				return gwCopy
			}),
		})
	}
	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) computeGatewayAcceptedCondition(ctx context.Context, gateway *gwv1.Gateway, effectiveGatewayClass *gwv1.GatewayClass) ([]*gatewayAcceptedCondition, error) {
	// 1. List all Gateways in the namespace whose GatewayClass is current effective class
	gatewayList := &gwv1.GatewayList{}
	if err := r.fctx.List(ctx, gatewayList, client.InNamespace(gateway.Namespace)); err != nil {
		log.Error().Msgf("Failed to list all gateways in namespace %s: %s", gateway.Namespace, err)
		return nil, err
	}

	// 2. Find the oldest Gateway in the namespace, if CreateTimestamp is equal, then sort by alphabet order asc.
	// If spec.GatewayClassName equals effectiveGatewayClass then it's a valid gateway
	// Otherwise, it's invalid
	validGateways := make([]*gwv1.Gateway, 0)
	//invalidGateways := make([]*gwv1.Gateway, 0)

	for _, gw := range gatewayList.Items {
		gw := gw // fix lint GO-LOOP-REF
		if string(gw.Spec.GatewayClassName) == effectiveGatewayClass.Name {
			validGateways = append(validGateways, &gw)
		}
		//else {
		//	invalidGateways = append(invalidGateways, &gw)
		//}
	}

	//sort.Slice(validGateways, func(i, j int) bool {
	//	if validGateways[i].CreationTimestamp.Time.Equal(validGateways[j].CreationTimestamp.Time) {
	//		return client.ObjectKeyFromObject(validGateways[i]).String() < client.ObjectKeyFromObject(validGateways[j]).String()
	//	}
	//
	//	return validGateways[i].CreationTimestamp.Time.Before(validGateways[j].CreationTimestamp.Time)
	//})

	// 3. Set the oldest as Accepted and the rest are unaccepted
	acceptedStatusChangedGatewayConditions := make([]*gatewayAcceptedCondition, 0)

	for i := range gwutils.SortResources(validGateways) {
		if i == 0 {
			if !gwutils.IsAcceptedGateway(validGateways[i]) {
				acceptedStatusChangedGatewayConditions = append(acceptedStatusChangedGatewayConditions, &gatewayAcceptedCondition{
					gateway:   validGateways[i],
					condition: r.acceptedCondition(validGateways[i]),
				})
			}
		} else {
			if gwutils.IsAcceptedGateway(validGateways[i]) {
				acceptedStatusChangedGatewayConditions = append(acceptedStatusChangedGatewayConditions, &gatewayAcceptedCondition{
					gateway:   validGateways[i],
					condition: r.unacceptedCondition(validGateways[i]),
				})
			}
		}
	}

	// in case of effective GatewayClass changed or spec.GatewayClassName was changed
	//for i := range invalidGateways {
	//	if gwutils.IsAcceptedGateway(invalidGateways[i]) {
	//		//r.unacceptedCondition(invalidGateways[i])
	//		acceptedStatusChangedGatewayConditions = append(acceptedStatusChangedGatewayConditions, &gatewayAcceptedCondition{
	//			gateway:   invalidGateways[i],
	//			condition: r.unacceptedCondition(invalidGateways[i]),
	//		})
	//	}
	//}

	return acceptedStatusChangedGatewayConditions, nil
}

func (r *gatewayReconciler) getNodeIPs(ctx context.Context, svc *corev1.Service) []string {
	pods := &corev1.PodList{}
	if err := r.fctx.List(
		ctx,
		pods,
		client.InNamespace(svc.Namespace),
		client.MatchingLabelsSelector{
			Selector: labels.SelectorFromSet(svc.Spec.Selector),
		},
	); err != nil {
		log.Error().Msgf("Failed to get pods: %s", err)
		return nil
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

			log.Error().Msgf("Failed to get node %q: %s", pod.Spec.NodeName, err)
			return nil
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
			return nil
		}

		nodeIPs = ips
	}

	return nodeIPs
}

func (r *gatewayReconciler) updateGatewayAddresses(ctx context.Context, gateway *gwv1.Gateway) (ctrl.Result, error) {
	log.Debug().Msgf("[GW] 5.1")
	// 6. after all status of gateways in the namespace have been updated successfully
	//   list all gateways in the namespace and deploy/redeploy the effective one
	preActiveGateway, err := r.findPreActiveGatewayByNamespace(ctx, gateway.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Debug().Msgf("[GW] 5.2")
	if preActiveGateway == nil {
		log.Warn().Msgf("[GW] No active gateway found in namespace %s", gateway.Namespace)
		return ctrl.Result{}, nil
	}

	log.Debug().Msgf("[GW] 5.3")
	if !isSameGateway(activeGateways[gateway.Namespace], preActiveGateway) {
		log.Debug().Msgf("[GW] 5.3.1")
		result, err := r.applyGateway(preActiveGateway)
		log.Debug().Msgf("[GW] 5.3.2")
		if err != nil {
			log.Debug().Msgf("[GW] 5.3.3")
			return result, err
		}
		log.Debug().Msgf("[GW] 5.3.4")
		activeGateways[gateway.Namespace] = preActiveGateway
	}

	log.Debug().Msgf("[GW] 5.4")
	// 7. update addresses of Gateway status if any IP is allocated
	addresses := r.gatewayAddresses(ctx, preActiveGateway)

	log.Debug().Msgf("[GW] 5.5")
	condition, programmed := r.computeGatewayProgrammedCondition(ctx, preActiveGateway, addresses)

	log.Debug().Msgf("[GW] 5.6")
	r.fctx.StatusUpdater.Send(status.Update{
		Resource:       &gwv1.Gateway{},
		NamespacedName: client.ObjectKeyFromObject(preActiveGateway),
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			gw, ok := obj.(*gwv1.Gateway)
			if !ok {
				log.Error().Msgf("Unexpected object type %T", obj)
			}

			gwCopy := gw.DeepCopy()
			gwCopy.Status.Addresses = addresses
			metautil.SetStatusCondition(&gwCopy.Status.Conditions, condition)

			return gwCopy
		}),
	})

	log.Debug().Msgf("[GW] 5.7")
	if !programmed {
		log.Debug().Msgf("[GW] 5.7.1")
		log.Debug().Msgf("[GW] Requeue gateway %s/%s after 3 second", preActiveGateway.Namespace, preActiveGateway.Name)
		return ctrl.Result{RequeueAfter: 3 * time.Second}, nil
	}

	log.Debug().Msgf("[GW] 5.8")
	// if there's any previous active gateways and has been assigned addresses, clean it up
	gatewayList := &gwv1.GatewayList{}
	if err := r.fctx.List(ctx, gatewayList, client.InNamespace(preActiveGateway.Namespace)); err != nil {
		log.Error().Msgf("Failed to list all gateways in namespace %s: %s", preActiveGateway.Namespace, err)
		return ctrl.Result{}, err
	}

	log.Debug().Msgf("[GW] 5.9")
	for _, gw := range gatewayList.Items {
		gw := gw // fix lint GO-LOOP-REF
		if gw.Name != preActiveGateway.Name && len(gw.Status.Addresses) > 0 {
			r.fctx.StatusUpdater.Send(status.Update{
				Resource:       &gwv1.Gateway{},
				NamespacedName: client.ObjectKeyFromObject(&gw),
				Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
					gwy, ok := obj.(*gwv1.Gateway)
					if !ok {
						log.Error().Msgf("Unexpected object type %T", obj)
					}
					gwCopy := gwy.DeepCopy()
					gwCopy.Status.Addresses = nil

					return gwCopy
				}),
			})
		}
	}

	log.Debug().Msgf("[GW] 5.10")
	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) gatewayService(ctx context.Context, activeGateway *gwv1.Gateway) (*corev1.Service, error) {
	serviceName := gatewayServiceName(activeGateway)
	if serviceName == "" {
		log.Warn().Msgf("[GW] No supported service protocols for Gateway %s/%s, only TCP and UDP are supported now.", activeGateway.Namespace, activeGateway.Name)
		return nil, fmt.Errorf("no supported service protocols for Gateway %s/%s, only TCP and UDP are supported", activeGateway.Namespace, activeGateway.Name)
	}

	svc := &corev1.Service{}
	key := client.ObjectKey{
		Namespace: activeGateway.Namespace,
		Name:      serviceName,
	}
	if err := r.fctx.Get(ctx, key, svc); err != nil {
		return nil, err
	}

	return svc, nil
}

func (r *gatewayReconciler) gatewayAddresses(ctx context.Context, gw *gwv1.Gateway) []gwv1.GatewayStatusAddress {
	gwSvc, err := r.gatewayService(ctx, gw)
	if err != nil {
		log.Error().Msgf("Failed to get gateway service: %s", err)
		return nil
	}

	var addresses, hostnames []string

	switch gwSvc.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		for i := range gwSvc.Status.LoadBalancer.Ingress {
			switch {
			case len(gwSvc.Status.LoadBalancer.Ingress[i].IP) > 0:
				addresses = append(addresses, gwSvc.Status.LoadBalancer.Ingress[i].IP)
			case len(gwSvc.Status.LoadBalancer.Ingress[i].Hostname) > 0:
				if gwSvc.Status.LoadBalancer.Ingress[i].Hostname == "localhost" {
					addresses = append(addresses, "127.0.0.1")
				}
				hostnames = append(hostnames, gwSvc.Status.LoadBalancer.Ingress[i].Hostname)
			}
		}
	case corev1.ServiceTypeNodePort:
		addresses = append(addresses, r.getNodeIPs(ctx, gwSvc)...)
	default:
		return nil
	}

	var gwAddresses []gwv1.GatewayStatusAddress
	for i := range addresses {
		addr := gwv1.GatewayStatusAddress{
			Type:  ptr.To(gwv1.IPAddressType),
			Value: addresses[i],
		}
		gwAddresses = append(gwAddresses, addr)
	}

	for i := range hostnames {
		addr := gwv1.GatewayStatusAddress{
			Type:  ptr.To(gwv1.HostnameAddressType),
			Value: hostnames[i],
		}
		gwAddresses = append(gwAddresses, addr)
	}

	return gwAddresses
}

func gatewayServiceName(activeGateway *gwv1.Gateway) string {
	if hasTCP(activeGateway) {
		return fmt.Sprintf("fsm-gateway-%s-tcp", activeGateway.Namespace)
	}

	if hasUDP(activeGateway) {
		return fmt.Sprintf("fsm-gateway-%s-udp", activeGateway.Namespace)
	}

	return ""
}

func (r *gatewayReconciler) computeGatewayProgrammedCondition(ctx context.Context, gw *gwv1.Gateway, addresses []gwv1.GatewayStatusAddress) (metav1.Condition, bool) {
	deployment := r.gatewayDeployment(ctx, gw)

	if len(addresses) == 0 {
		defer r.recorder.Eventf(gw, corev1.EventTypeWarning, "Addresses", "No addresses have been assigned to the Gateway")

		return metav1.Condition{
			Type:               string(gwv1.GatewayConditionProgrammed),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: gw.Generation,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             string(gwv1.GatewayReasonAddressNotAssigned),
			Message:            "No addresses have been assigned to the Gateway",
		}, false
	}

	if deployment == nil || deployment.Status.AvailableReplicas == 0 {
		defer r.recorder.Eventf(gw, corev1.EventTypeWarning, "Unavailable", "Gateway Deployment replicas unavailable")

		return metav1.Condition{
			Type:               string(gwv1.GatewayConditionProgrammed),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: gw.Generation,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             string(gwv1.GatewayReasonNoResources),
			Message:            "Deployment replicas unavailable",
		}, false
	}

	defer r.recorder.Eventf(gw, corev1.EventTypeNormal, "Programmed", "Address assigned to the Gateway, Gateway is programmed")

	return metav1.Condition{
		Type:               string(gwv1.GatewayConditionProgrammed),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: gw.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1.GatewayConditionProgrammed),
		Message:            fmt.Sprintf("Address assigned to the Gateway, %d/%d Deployment replicas available", deployment.Status.AvailableReplicas, deployment.Status.Replicas),
	}, true
}

func (r *gatewayReconciler) gatewayDeployment(ctx context.Context, gw *gwv1.Gateway) *appsv1.Deployment {
	deployment := &appsv1.Deployment{}
	key := types.NamespacedName{
		Namespace: gw.Namespace,
		Name:      fmt.Sprintf("fsm-gateway-%s", gw.Namespace),
	}

	if err := r.fctx.Get(ctx, key, deployment); err != nil {
		log.Error().Msgf("Failed to get deployment %s: %s", key.String(), err)
		return nil
	}

	return deployment
}

func (r *gatewayReconciler) updateListenerStatus(_ context.Context, gateway *gwv1.Gateway) (ctrl.Result, error) {
	if listenerStatus := r.computeListenerStatuses(gateway); len(listenerStatus) > 0 {
		opts := cmpopts.IgnoreFields(metav1.Condition{}, "LastTransitionTime")
		if cmp.Equal(gateway.Status.Listeners, listenerStatus, opts) {
			log.Debug().Msgf("[GW] listener status unchanged, bypassing update")
			return ctrl.Result{}, nil
		}

		r.fctx.StatusUpdater.Send(status.Update{
			Resource:       &gwv1.Gateway{},
			NamespacedName: client.ObjectKeyFromObject(gateway),
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				gw, ok := obj.(*gwv1.Gateway)
				if !ok {
					log.Error().Msgf("Unexpected object type %T", obj)
				}
				gwCopy := gw.DeepCopy()
				gwCopy.Status.Listeners = listenerStatus

				defer r.recorder.Eventf(gwCopy, corev1.EventTypeNormal, "Listeners", "Status of Listeners updated")

				return gwCopy
			}),
		})
	}

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) computeListenerStatuses(gateway *gwv1.Gateway) []gwv1.ListenerStatus {
	existingListenerStatus := make(map[gwv1.SectionName]gwv1.ListenerStatus)
	for _, s := range gateway.Status.Listeners {
		existingListenerStatus[s.Name] = s
	}

	listenerStatus := make([]gwv1.ListenerStatus, 0)
	for _, listener := range gateway.Spec.Listeners {
		s, ok := existingListenerStatus[listener.Name]
		if ok {
			// update existing status
			programmedConditionExists := false
			acceptedConditionExists := false
			for _, cond := range s.Conditions {
				if cond.Type == string(gwv1.ListenerConditionProgrammed) {
					programmedConditionExists = true
				}
				if cond.Type == string(gwv1.ListenerConditionAccepted) {
					acceptedConditionExists = true
				}
			}

			if !programmedConditionExists {
				metautil.SetStatusCondition(&s.Conditions, metav1.Condition{
					Type:               string(gwv1.ListenerConditionProgrammed),
					Status:             metav1.ConditionFalse,
					ObservedGeneration: gateway.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1.ListenerReasonInvalid),
					Message:            fmt.Sprintf("Invalid listener %q[:%d]", listener.Name, listener.Port),
				})
			}

			if !acceptedConditionExists {
				metautil.SetStatusCondition(&s.Conditions, metav1.Condition{
					Type:               string(gwv1.ListenerConditionAccepted),
					Status:             metav1.ConditionTrue,
					ObservedGeneration: gateway.Generation,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             string(gwv1.ListenerReasonAccepted),
					Message:            fmt.Sprintf("listener %q[:%d] is accepted.", listener.Name, listener.Port),
				})
			}
		} else {
			// create new status
			s = gwv1.ListenerStatus{Name: listener.Name}
			kinds, conditions := supportedRouteGroupKinds(gateway, listener)

			if len(conditions) == 0 {
				s.Conditions = []metav1.Condition{
					{
						Type:               string(gwv1.ListenerConditionAccepted),
						Status:             metav1.ConditionTrue,
						ObservedGeneration: gateway.Generation,
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             string(gwv1.ListenerReasonAccepted),
						Message:            fmt.Sprintf("listener %q[:%d] is accepted.", listener.Name, listener.Port),
					},
					{
						Type:               string(gwv1.ListenerConditionProgrammed),
						Status:             metav1.ConditionTrue,
						ObservedGeneration: gateway.Generation,
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             string(gwv1.ListenerReasonProgrammed),
						Message:            fmt.Sprintf("Valid listener %q[:%d]", listener.Name, listener.Port),
					},
				}
			} else {
				s.Conditions = conditions
			}

			s.SupportedKinds = kinds
		}

		listenerStatus = append(listenerStatus, s)
	}
	return listenerStatus
}

func supportedRouteGroupKinds(gateway *gwv1.Gateway, listener gwv1.Listener) ([]gwv1.RouteGroupKind, []metav1.Condition) {
	if len(listener.AllowedRoutes.Kinds) == 0 {
		switch listener.Protocol {
		case gwv1.HTTPProtocolType, gwv1.HTTPSProtocolType:
			return []gwv1.RouteGroupKind{
				{
					Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
					Kind:  constants.GatewayAPIHTTPRouteKind,
				},
				{
					Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
					Kind:  constants.GatewayAPIGRPCRouteKind,
				},
			}, nil
		case gwv1.TLSProtocolType:
			return []gwv1.RouteGroupKind{
				{
					Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
					Kind:  constants.GatewayAPITLSRouteKind,
				},
				{
					Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
					Kind:  constants.GatewayAPITCPRouteKind,
				},
			}, nil
		case gwv1.TCPProtocolType:
			return []gwv1.RouteGroupKind{
				{
					Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
					Kind:  constants.GatewayAPITCPRouteKind,
				},
			}, nil
		case gwv1.UDPProtocolType:
			return []gwv1.RouteGroupKind{
				{
					Group: gwutils.GroupPointer(constants.GatewayAPIGroup),
					Kind:  constants.GatewayAPIUDPRouteKind,
				},
			}, nil
		}
	}

	kinds := make([]gwv1.RouteGroupKind, 0)
	conditions := make([]metav1.Condition, 0)

	for _, routeKind := range listener.AllowedRoutes.Kinds {
		if routeKind.Group != nil && *routeKind.Group != constants.GatewayAPIGroup {
			conditions = append(conditions, metav1.Condition{
				Type:               string(gwv1.ListenerConditionResolvedRefs),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: gateway.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1.ListenerReasonInvalidRouteKinds),
				Message:            fmt.Sprintf("Group %q is not supported, group must be %q", *routeKind.Group, gwv1.GroupName),
			})
			continue
		}

		if routeKind.Kind != constants.GatewayAPIHTTPRouteKind &&
			routeKind.Kind != constants.GatewayAPITLSRouteKind &&
			routeKind.Kind != constants.GatewayAPIGRPCRouteKind &&
			routeKind.Kind != constants.GatewayAPITCPRouteKind &&
			routeKind.Kind != constants.GatewayAPIUDPRouteKind {
			conditions = append(conditions, metav1.Condition{
				Type:               string(gwv1.ListenerConditionResolvedRefs),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: gateway.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1.ListenerReasonInvalidRouteKinds),
				Message:            fmt.Sprintf("Kind %q is not supported, kind must be %q, %q, %q, %q or %q", routeKind.Kind, constants.GatewayAPIHTTPRouteKind, constants.GatewayAPIGRPCRouteKind, constants.GatewayAPITLSRouteKind, constants.GatewayAPITCPRouteKind, constants.GatewayAPIUDPRouteKind),
			})
			continue
		}

		if routeKind.Kind == constants.GatewayAPIHTTPRouteKind && listener.Protocol != gwv1.HTTPProtocolType && listener.Protocol != gwv1.HTTPSProtocolType {
			conditions = append(conditions, metav1.Condition{
				Type:               string(gwv1.ListenerConditionResolvedRefs),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: gateway.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1.ListenerReasonInvalidRouteKinds),
				Message:            fmt.Sprintf("HTTPRoutes are incompatible with listener protocol %q", listener.Protocol),
			})
			continue
		}

		if routeKind.Kind == constants.GatewayAPIGRPCRouteKind && listener.Protocol != gwv1.HTTPProtocolType && listener.Protocol != gwv1.HTTPSProtocolType {
			conditions = append(conditions, metav1.Condition{
				Type:               string(gwv1.ListenerConditionResolvedRefs),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: gateway.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1.ListenerReasonInvalidRouteKinds),
				Message:            fmt.Sprintf("GRPCRoutes are incompatible with listener protocol %q", listener.Protocol),
			})
			continue
		}

		if routeKind.Kind == constants.GatewayAPITLSRouteKind && listener.Protocol != gwv1.TLSProtocolType {
			conditions = append(conditions, metav1.Condition{
				Type:               string(gwv1.ListenerConditionResolvedRefs),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: gateway.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1.ListenerReasonInvalidRouteKinds),
				Message:            fmt.Sprintf("TLSRoutes are incompatible with listener protocol %q", listener.Protocol),
			})
			continue
		}

		if routeKind.Kind == constants.GatewayAPITCPRouteKind && listener.Protocol != gwv1.TCPProtocolType && listener.Protocol != gwv1.TLSProtocolType {
			conditions = append(conditions, metav1.Condition{
				Type:               string(gwv1.ListenerConditionResolvedRefs),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: gateway.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1.ListenerReasonInvalidRouteKinds),
				Message:            fmt.Sprintf("TCPRoutes are incompatible with listener protocol %q", listener.Protocol),
			})
			continue
		}

		if routeKind.Kind == constants.GatewayAPIUDPRouteKind && listener.Protocol != gwv1.UDPProtocolType {
			conditions = append(conditions, metav1.Condition{
				Type:               string(gwv1.ListenerConditionResolvedRefs),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: gateway.Generation,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             string(gwv1.ListenerReasonInvalidRouteKinds),
				Message:            fmt.Sprintf("UDPRoutes are incompatible with listener protocol %q", listener.Protocol),
			})
			continue
		}

		kinds = append(kinds, gwv1.RouteGroupKind{
			Group: routeKind.Group,
			Kind:  routeKind.Kind,
		})
	}

	return kinds, conditions
}

func (r *gatewayReconciler) findPreActiveGatewayByNamespace(ctx context.Context, namespace string) (*gwv1.Gateway, error) {
	gatewayList := &gwv1.GatewayList{}
	if err := r.fctx.List(ctx, gatewayList, client.InNamespace(namespace)); err != nil {
		log.Error().Msgf("Failed to list all gateways in namespace %s: %s", namespace, err)
		return nil, err
	}

	for _, gw := range gatewayList.Items {
		gw := gw // fix lint GO-LOOP-REF
		if gwutils.IsPreActiveGateway(&gw) {
			return &gw, nil
		}
	}

	return nil, nil
}

func isSameGateway(oldGateway, newGateway *gwv1.Gateway) bool {
	return equality.Semantic.DeepEqual(oldGateway, newGateway)
}

func (r *gatewayReconciler) applyGateway(gateway *gwv1.Gateway) (ctrl.Result, error) {
	mc := r.fctx.Configurator

	result, err := r.deriveCodebases(gateway, mc)
	if err != nil {
		return result, err
	}

	result, err = r.updateConfig(gateway, mc)
	if err != nil {
		return result, err
	}

	return r.deployGateway(gateway, mc)
}

func (r *gatewayReconciler) deriveCodebases(gw *gwv1.Gateway, _ configurator.Configurator) (ctrl.Result, error) {
	gwPath := utils.GatewayCodebasePath(gw.Namespace)
	parentPath := utils.GetDefaultGatewaysPath()
	if err := r.fctx.RepoClient.DeriveCodebase(gwPath, parentPath); err != nil {
		defer r.recorder.Eventf(gw, corev1.EventTypeWarning, "Codebase", "Failed to derive codebase of gateway: %s", err)

		return ctrl.Result{RequeueAfter: 1 * time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) updateConfig(gw *gwv1.Gateway, _ configurator.Configurator) (ctrl.Result, error) {
	// TODO: update pipy repo
	// defer r.recorder.Eventf(gw, corev1.EventTypeWarning, "Repo", "Failed to update repo config of gateway: %s", err)
	defer r.recorder.Eventf(gw, corev1.EventTypeNormal, "Repo", "Update repo config of gateway successfully")
	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) deployGateway(gw *gwv1.Gateway, mc configurator.Configurator) (ctrl.Result, error) {
	actionConfig := helm.ActionConfig(gw.Namespace, log.Debug().Msgf)

	templateClient := helm.TemplateClient(
		actionConfig,
		fmt.Sprintf("fsm-gateway-%s", gw.Namespace),
		gw.Namespace,
		r.kubeVersionForTemplate(),
	)
	if ctrlResult, err := helm.RenderChart(templateClient, gw, chartSource, mc, r.fctx.Client, r.fctx.Scheme, r.resolveValues); err != nil {
		defer r.recorder.Eventf(gw, corev1.EventTypeWarning, "Deploy", "Failed to deploy gateway: %s", err)
		return ctrlResult, err
	}
	defer r.recorder.Eventf(gw, corev1.EventTypeNormal, "Deploy", "Deploy gateway successfully")

	return ctrl.Result{}, nil
}

func (r *gatewayReconciler) kubeVersionForTemplate() *chartutil.KubeVersion {
	if version.IsEndpointSliceEnabled(r.fctx.KubeClient) {
		return constants.KubeVersion121
	}

	return constants.KubeVersion119
}

func (r *gatewayReconciler) resolveValues(object metav1.Object, mc configurator.Configurator) (map[string]interface{}, error) {
	gateway, ok := object.(*gwv1.Gateway)
	if !ok {
		return nil, fmt.Errorf("object %v is not type of *gwv1.Gateway", object)
	}

	log.Debug().Msgf("[GW] Resolving Values ...")

	//gwBytes, err := ghodssyaml.Marshal(&gatewayValues{
	//	Gateway:   gateway,
	//	Listeners: gwutils.GetValidListenersForGateway(gateway),
	//})
	//if err != nil {
	//	return nil, fmt.Errorf("convert Gateway to yaml, err = %v", err)
	//}
	//log.Debug().Msgf("\n\nGATEWAY VALUES YAML:\n\n\n%s\n\n", string(gwBytes))
	//gwValues, err := chartutil.ReadValues(gwBytes)
	//if err != nil {
	//	return nil, err
	//}

	// these values are from MeshConfig and Gateway resource, it will not be overridden by values from ParametersRef
	gwBytes, err := ghodssyaml.Marshal(map[string]interface{}{
		"fsm": map[string]interface{}{
			"fsmNamespace": mc.GetFSMNamespace(),
			"meshName":     r.fctx.MeshName,
			"gateway": map[string]interface{}{
				"namespace":      gateway.Namespace,
				"listeners":      listenersForTemplate(gateway),
				"infrastructure": infraForTemplate(gateway),
				"logLevel":       mc.GetFSMGatewayLogLevel(),
			},
			"image": map[string]interface{}{
				"registry":   mc.GetImageRegistry(),
				"tag":        mc.GetImageTag(),
				"pullPolicy": mc.GetImagePullPolicy(),
			},
		},
		"hasTCP": hasTCP(gateway),
		"hasUDP": hasUDP(gateway),
	})
	if err != nil {
		return nil, fmt.Errorf("convert Gateway to yaml, err = %v", err)
	}
	log.Debug().Msgf("\n\nGATEWAY VALUES YAML:\n\n\n%s\n\n", string(gwBytes))
	gwValues, err := chartutil.ReadValues(gwBytes)
	if err != nil {
		return nil, err
	}

	gatewayValues := gwValues.AsMap()

	//overrides := []string{
	//	fmt.Sprintf("fsm.image.registry=%s", mc.GetImageRegistry()),
	//	fmt.Sprintf("fsm.image.tag=%s", mc.GetImageTag()),
	//	fmt.Sprintf("fsm.image.pullPolicy=%s", mc.GetImagePullPolicy()),
	//	fmt.Sprintf("fsm.fsmNamespace=%s", mc.GetFSMNamespace()),
	//	fmt.Sprintf("fsm.gateway.logLevel=%s", mc.GetFSMGatewayLogLevel()),
	//	fmt.Sprintf("fsm.meshName=%s", r.fctx.MeshName),
	//	fmt.Sprintf("hasTCP=%t", hasTCP(gateway)),
	//	fmt.Sprintf("hasUDP=%t", hasUDP(gateway)),
	//}
	//
	//for _, ov := range overrides {
	//	if err := strvals.ParseInto(ov, gatewayValues); err != nil {
	//		return nil, err
	//	}
	//}

	parameterValues, err := r.resolveParameterValues(gateway)
	if err != nil {
		log.Error().Msgf("Failed to resolve parameter values from ParametersRef: %s, it doesn't take effect", err)
		return gatewayValues, nil
	}

	if parameterValues == nil {
		return gatewayValues, nil
	}

	// gateway values take precedence over parameter values, means the values from MeshConfig override the values from ParametersRef
	// see the overrides variables for a complete list of values
	return chartutil.CoalesceTables(parameterValues, gatewayValues), nil
}

func infraForTemplate(gateway *gwv1.Gateway) map[string]map[gwv1.AnnotationKey]gwv1.AnnotationValue {
	infra := map[string]map[gwv1.AnnotationKey]gwv1.AnnotationValue{
		"annotations": {},
		"labels":      {},
	}

	if gateway.Spec.Infrastructure != nil {
		if len(gateway.Spec.Infrastructure.Annotations) > 0 {
			infra["annotations"] = gateway.Spec.Infrastructure.Annotations
		}
		if len(gateway.Spec.Infrastructure.Labels) > 0 {
			infra["labels"] = gateway.Spec.Infrastructure.Labels
		}
	}

	return infra
}

func listenersForTemplate(gateway *gwv1.Gateway) []Listener {
	listeners := make([]Listener, 0)
	for _, l := range gwutils.GetValidListenersForGateway(gateway) {
		listeners = append(listeners, Listener{
			Name:     l.Listener.Name,
			Port:     l.Listener.Port,
			Protocol: l.Listener.Protocol,
		})
	}

	return listeners
}

func (r *gatewayReconciler) resolveParameterValues(gateway *gwv1.Gateway) (map[string]interface{}, error) {
	if gateway.Spec.Infrastructure == nil {
		return nil, nil
	}

	if gateway.Spec.Infrastructure.ParametersRef == nil {
		return nil, nil
	}

	paramRef := gateway.Spec.Infrastructure.ParametersRef
	if paramRef.Group != corev1.GroupName {
		return nil, nil
	}

	if paramRef.Kind != constants.KubernetesConfigMapKind {
		return nil, nil
	}

	cm := &corev1.ConfigMap{}
	key := types.NamespacedName{
		Namespace: gateway.Namespace,
		Name:      paramRef.Name,
	}

	if err := r.fctx.Get(context.TODO(), key, cm); err != nil {
		return nil, fmt.Errorf("failed to get Configmap %s: %s", key, err)
	}

	if len(cm.Data) == 0 {
		return nil, fmt.Errorf("configmap %q has no data", key)
	}

	valuesYaml, ok := cm.Data["values.yaml"]
	if !ok {
		return nil, fmt.Errorf("configmap %q has no values.yaml", key)
	}

	log.Debug().Msgf("[GW] values.yaml from ConfigMap %s: \n%s\n", key.String(), valuesYaml)

	paramsMap := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(valuesYaml), &paramsMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal values.yaml of Configmap %s: %s", key, err)
	}

	log.Debug().Msgf("[GW] values parsed from values.yaml: %v", paramsMap)

	return paramsMap, nil
}

func (r *gatewayReconciler) acceptedCondition(gateway *gwv1.Gateway) metav1.Condition {
	return metav1.Condition{
		Type:               string(gwv1.GatewayConditionAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: gateway.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             string(gwv1.GatewayReasonAccepted),
		Message:            fmt.Sprintf("Gateway %s/%s is accepted.", gateway.Namespace, gateway.Name),
	}
}

func (r *gatewayReconciler) unacceptedCondition(gateway *gwv1.Gateway) metav1.Condition {
	return metav1.Condition{
		Type:               string(gwv1.GatewayConditionAccepted),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: gateway.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "Unaccepted",
		Message:            fmt.Sprintf("Gateway %s/%s is not accepted as it's not the oldest one in namespace %q.", gateway.Namespace, gateway.Name, gateway.Namespace),
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *gatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&gwv1.Gateway{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			gateway, ok := obj.(*gwv1.Gateway)
			if !ok {
				log.Error().Msgf("unexpected object type %T", obj)
				return false
			}

			gatewayClass := &gwv1.GatewayClass{}
			key := types.NamespacedName{Name: string(gateway.Spec.GatewayClassName)}
			if err := r.fctx.Get(context.TODO(), key, gatewayClass); err != nil {
				log.Error().Msgf("failed to get gatewayclass %s", gateway.Spec.GatewayClassName)
				return false
			}

			if gatewayClass.Spec.ControllerName != constants.GatewayController {
				log.Warn().Msgf("class controller of Gateway %s/%s is not %s", gateway.Namespace, gateway.Name, constants.GatewayController)
				return false
			}

			return true
		}))).
		Watches(
			&gwv1.GatewayClass{},
			handler.EnqueueRequestsFromMapFunc(r.gatewayClassToGateways),
			builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
				gatewayClass, ok := obj.(*gwv1.GatewayClass)
				if !ok {
					log.Error().Msgf("unexpected object type: %T", obj)
					return false
				}

				return gatewayClass.Spec.ControllerName == constants.GatewayController
			})),
		).
		Watches(
			&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.configMapToGateways),
		).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.secretToGateways),
		).
		Complete(r); err != nil {
		return err
	}

	return addGatewayIndexers(context.TODO(), mgr)
}

func (r *gatewayReconciler) gatewayClassToGateways(ctx context.Context, obj client.Object) []reconcile.Request {
	gatewayClass, ok := obj.(*gwv1.GatewayClass)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", obj)
		return nil
	}

	if gwutils.IsEffectiveGatewayClass(gatewayClass) {
		c := r.fctx.Manager.GetCache()
		gateways := &gwv1.GatewayList{}
		err := c.List(ctx, gateways, &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(constants.ClassGatewayIndex, gatewayClass.Name),
		})
		//var gateways gwv1.GatewayList
		//if err := r.fctx.List(ctx, &gateways); err != nil {
		if err != nil {
			log.Error().Msgf("error listing gateways: %s", err)
			return nil
		}

		var reconciles []reconcile.Request
		for _, gw := range gateways.Items {
			gw := gw
			//if string(gw.Spec.GatewayClassName) == gatewayClass.GetName() {
			if gwutils.IsActiveGateway(&gw) {
				reconciles = append(reconciles, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: gw.Namespace,
						Name:      gw.Name,
					},
				})
			}
		}

		return reconciles
	}

	return nil
}

func (r *gatewayReconciler) configMapToGateways(ctx context.Context, object client.Object) []reconcile.Request {
	cm, ok := object.(*corev1.ConfigMap)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", object)
		return nil
	}

	c := r.fctx.Manager.GetCache()
	gateways := &gwv1.GatewayList{}
	err := c.List(ctx, gateways, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ConfigMapGatewayIndex, client.ObjectKeyFromObject(cm).String()),
		Namespace:     cm.Namespace,
	})
	//err := r.fctx.List(ctx, gateways, client.InNamespace(cm.Namespace))
	if err != nil {
		log.Error().Msgf("error listing gateways: %s", err)
		return nil
	}

	if len(gateways.Items) == 0 {
		return nil
	}

	reconciles := make([]reconcile.Request, 0)
	for _, gw := range gateways.Items {
		gw := gw
		if gwutils.IsActiveGateway(&gw) {
			reconciles = append(reconciles, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: gw.Namespace,
					Name:      gw.Name,
				},
			})
		}
		//if gw.Spec.Infrastructure == nil {
		//	continue
		//}
		//
		//if gw.Spec.Infrastructure.ParametersRef == nil {
		//	continue
		//}
		//
		//paramRef := gw.Spec.Infrastructure.ParametersRef
		//if paramRef.Name == cm.Name && paramRef.Group == corev1.GroupName && paramRef.Kind == constants.KubernetesConfigMapKind {
		//	reconciles = append(reconciles, reconcile.Request{
		//		NamespacedName: types.NamespacedName{
		//			Namespace: gw.Namespace,
		//			Name:      gw.Name,
		//		},
		//	})
		//}
	}

	return reconciles
}

func (r *gatewayReconciler) secretToGateways(ctx context.Context, object client.Object) []reconcile.Request {
	secret, ok := object.(*corev1.Secret)
	if !ok {
		log.Error().Msgf("unexpected object type: %T", object)
		return nil
	}

	c := r.fctx.Manager.GetCache()
	gateways := &gwv1.GatewayList{}
	err := c.List(ctx, gateways, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.SecretGatewayIndex, client.ObjectKeyFromObject(secret).String()),
	})
	if err != nil {
		log.Error().Msgf("error listing gateways: %s", err)
		return nil
	}

	if len(gateways.Items) == 0 {
		return nil
	}

	reconciles := make([]reconcile.Request, 0)
	for _, gw := range gateways.Items {
		gw := gw
		if gwutils.IsActiveGateway(&gw) {
			reconciles = append(reconciles, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: gw.Namespace,
					Name:      gw.Name,
				},
			})
		}
	}

	return reconciles
}

func addGatewayIndexers(ctx context.Context, mgr manager.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.Gateway{}, constants.SecretGatewayIndex, secretGatewayIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.Gateway{}, constants.ConfigMapGatewayIndex, configMapGatewayIndexFunc); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &gwv1.Gateway{}, constants.ClassGatewayIndex, func(obj client.Object) []string {
		gateway := obj.(*gwv1.Gateway)
		return []string{string(gateway.Spec.GatewayClassName)}
	}); err != nil {
		return err
	}

	return nil
}

func secretGatewayIndexFunc(obj client.Object) []string {
	gateway := obj.(*gwv1.Gateway)
	var secretReferences []string
	for _, listener := range gateway.Spec.Listeners {
		if listener.Protocol != gwv1.TLSProtocolType && listener.Protocol != gwv1.HTTPSProtocolType {
			continue
		}

		if listener.TLS == nil || *listener.TLS.Mode != gwv1.TLSModeTerminate {
			continue
		}

		for _, cert := range listener.TLS.CertificateRefs {
			if *cert.Kind == constants.KubernetesSecretKind {
				secretReferences = append(secretReferences,
					types.NamespacedName{
						Namespace: gwutils.NamespaceDerefOr(cert.Namespace, gateway.Namespace),
						Name:      string(cert.Name),
					}.String(),
				)
			}
		}

		if listener.TLS.FrontendValidation != nil {
			for _, ca := range listener.TLS.FrontendValidation.CACertificateRefs {
				if ca.Kind == constants.KubernetesSecretKind {
					secretReferences = append(secretReferences,
						types.NamespacedName{
							Namespace: gwutils.NamespaceDerefOr(ca.Namespace, gateway.Namespace),
							Name:      string(ca.Name),
						}.String(),
					)
				}
			}
		}
	}

	return secretReferences
}

func configMapGatewayIndexFunc(obj client.Object) []string {
	gateway := obj.(*gwv1.Gateway)
	var cmRefs []string

	// check against listeners
	for _, listener := range gateway.Spec.Listeners {
		if listener.Protocol != gwv1.TLSProtocolType && listener.Protocol != gwv1.HTTPSProtocolType {
			continue
		}

		if listener.TLS == nil || *listener.TLS.Mode != gwv1.TLSModeTerminate {
			continue
		}

		if listener.TLS.FrontendValidation == nil {
			continue
		}

		for _, ca := range listener.TLS.FrontendValidation.CACertificateRefs {
			if ca.Kind == constants.KubernetesConfigMapKind {
				cmRefs = append(cmRefs,
					types.NamespacedName{
						Namespace: gwutils.NamespaceDerefOr(ca.Namespace, gateway.Namespace),
						Name:      string(ca.Name),
					}.String(),
				)
			}
		}
	}

	// check against infrastructure ParametersRef
	if gateway.Spec.Infrastructure != nil && gateway.Spec.Infrastructure.ParametersRef != nil {
		parametersRef := gateway.Spec.Infrastructure.ParametersRef
		if parametersRef.Kind == constants.KubernetesConfigMapKind {
			cmRefs = append(cmRefs,
				types.NamespacedName{
					Namespace: gateway.Namespace,
					Name:      string(parametersRef.Name),
				}.String(),
			)
		}
	}

	return cmRefs
}
