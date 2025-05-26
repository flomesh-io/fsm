package ingress

import (
	"context"
	"errors"
	"fmt"
	"net"

	whtypes "github.com/flomesh-io/fsm/pkg/webhook/types"

	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/flomesh-io/fsm/pkg/webhook"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/webhook/builder"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type ClusterWebhook struct {
	webhook.DefaultWebhook
}

func NewClusterWebhook(cfg *whtypes.RegisterConfig) whtypes.Register {
	r := &ClusterWebhook{
		DefaultWebhook: webhook.DefaultWebhook{
			RegisterConfig: cfg,
			Client:         cfg.Manager.GetClient(),
		},
	}

	if blder, err := builder.WebhookConfigurationManagedBy(cfg.Manager).
		For(&mcsv1alpha1.Cluster{}).
		WithWebhookServiceName(cfg.WebhookSvcName).
		WithWebhookServiceNamespace(cfg.WebhookSvcNs).
		WithCABundle(cfg.CaBundle).
		Complete(); err != nil {
		return nil
	} else {
		r.CfgBuilder = blder
	}

	return r
}

func (r *ClusterWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, obj)
}

func (r *ClusterWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return r.doValidation(ctx, newObj)
}

func (r *ClusterWebhook) doValidation(_ context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	c, ok := obj.(*mcsv1alpha1.Cluster)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}

	host := c.Spec.GatewayHost
	if host == "" {
		return nil, errors.New("GatewayHost is required in OutCluster mode")
	}

	if c.Spec.Kubeconfig == "" {
		return nil, fmt.Errorf("kubeconfig must be set in OutCluster mode")
	}

	//if c.Name == "local" {
	//	return errors.New("Cluster Name 'local' is reserved for InCluster Mode ONLY, please change the cluster name")
	//}

	isDNSName := false
	if ipErrs := validation.IsValidIP(field.NewPath(""), host); len(ipErrs) > 0 {
		// Not IPv4 address
		log.Warn().Msgf("%q is NOT a valid IP address: %v", host, ipErrs)
		if dnsErrs := validation.IsDNS1123Subdomain(host); len(dnsErrs) > 0 {
			// Not valid DNS domain name
			return nil, fmt.Errorf("invalid DNS name %q: %v", host, dnsErrs)
		}

		// is DNS name
		isDNSName = true
	}

	var gwIP net.IP
	if isDNSName {
		ipAddr, err := net.ResolveIPAddr("ip", host)
		if err != nil {
			return nil, fmt.Errorf("%q cannot be resolved to IP", host)
		}
		log.Debug().Msgf("%q is resolved to IP: %s", host, ipAddr.IP)
		gwIP = ipAddr.IP
	} else {
		gwIP = net.ParseIP(host)
	}

	if gwIP == nil {
		return nil, fmt.Errorf("%q cannot be resolved to an IP address", host)
	}

	if gwIP.IsLoopback() || gwIP.IsUnspecified() {
		return nil, fmt.Errorf("gateway Host %s is resolved to Loopback IP or Unspecified", host)
	}

	port := int(c.Spec.GatewayPort)
	if errs := validation.IsValidPortNum(port); len(errs) > 0 {
		return nil, fmt.Errorf("invalid port number %d: %v", c.Spec.GatewayPort, errs)
	}

	return nil, nil
}
