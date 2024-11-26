package driver

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/models"
	"github.com/flomesh-io/fsm/pkg/sidecar/v1/driver"
	bootstrap2 "github.com/flomesh-io/fsm/pkg/sidecar/v1/providers/pipy/bootstrap"
	"github.com/flomesh-io/fsm/pkg/version"
)

// This will read an existing pipy bootstrap config, and create a new copy by changing the NodeID, and certificates.
func createPipyBootstrapFromExisting(ctx *driver.InjectorContext, newBootstrapSecretName, oldBootstrapSecretName, namespace string, cert *certificate.Certificate) (*corev1.Secret, error) {
	existing, err := ctx.KubeClient.CoreV1().Secrets(namespace).Get(context.Background(), oldBootstrapSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	configBytes := existing.Data[bootstrap2.PipyBootstrapConfigFile]
	config := bootstrap2.Builder{}
	if err = json.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("error unmarshalling pipy bootstrap config: %w", err)
	}
	config.NodeID = cert.GetCommonName().String()

	bootstrapConfig, err := json.MarshalIndent(config, "", "")
	if err != nil {
		return nil, err
	}

	return marshalAndSaveBootstrap(ctx, newBootstrapSecretName, namespace, bootstrapConfig, cert)
}

func createPipyBootstrapConfig(ctx *driver.InjectorContext, name, namespace, fsmNamespace string, cert *certificate.Certificate, originalHealthProbes models.HealthProbes) (*corev1.Secret, error) {
	builder := bootstrap2.Builder{
		NodeID: cert.GetCommonName().String(),

		RepoHost: fmt.Sprintf("%s.%s.svc.cluster.local", constants.FSMControllerName, fsmNamespace),
		RepoPort: ctx.Configurator.GetProxyServerPort(),

		// OriginalHealthProbes stores the path and port for liveness, readiness, and startup health probes as initially
		// defined on the Pod Spec.
		OriginalHealthProbes: originalHealthProbes,

		TLSMinProtocolVersion: ctx.Configurator.GetMeshConfig().Spec.Sidecar.TLSMinProtocolVersion,
		TLSMaxProtocolVersion: ctx.Configurator.GetMeshConfig().Spec.Sidecar.TLSMaxProtocolVersion,
		CipherSuites:          ctx.Configurator.GetMeshConfig().Spec.Sidecar.CipherSuites,
		ECDHCurves:            ctx.Configurator.GetMeshConfig().Spec.Sidecar.ECDHCurves,
	}
	bootstrapConfig, err := json.MarshalIndent(builder, "", "")
	if err != nil {
		return nil, err
	}

	return marshalAndSaveBootstrap(ctx, name, namespace, bootstrapConfig, cert)
}

func marshalAndSaveBootstrap(ctx *driver.InjectorContext, name, namespace string, config []byte, cert *certificate.Certificate) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
				constants.FSMAppInstanceLabelKey: ctx.MeshName,
				constants.FSMAppVersionLabelKey:  version.Version,
			},
		},
		Data: map[string][]byte{
			bootstrap2.PipyBootstrapConfigFile: config,
			bootstrap2.PipyRepoCACertFile:      cert.GetTrustedCAs(),
			bootstrap2.PipyRepoCertFile:        cert.GetCertificateChain(),
			bootstrap2.PipyRepoKeyFile:         cert.GetPrivateKey(),
		},
	}

	log.Debug().Msgf("Creating bootstrap config for Pipy: name=%s, namespace=%s", name, namespace)
	return ctx.KubeClient.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
}
