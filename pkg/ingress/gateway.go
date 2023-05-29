package ingress

import (
	"context"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/announcements"
	configv1alpha2 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/k8s/events"

	"github.com/flomesh-io/fsm/pkg/certificate"
)

// provisionIngressGatewayCert does the following:
//  1. If an ingress gateway certificate spec is specified in the MeshConfig resource, issues a certificate
//     for it and stores it in the referenced secret.
//  2. Starts a goroutine to watch for changes to the MeshConfig resource and certificate rotation, and
//     updates/removes the certificate and secret as necessary.
func (c *client) provisionIngressGatewayCert(stop <-chan struct{}) error {
	defaultCertSpec := c.cfg.GetMeshConfig().Spec.Certificate.IngressGateway
	if defaultCertSpec != nil {
		// Issue a certificate for the default certificate spec
		if err := c.createAndStoreGatewayCert(*defaultCertSpec); err != nil {
			return fmt.Errorf("Error provisioning default ingress gateway cert: %w", err)
		}
	}

	// Initialize a watcher to watch for CREATE/UPDATE/DELETE on the ingress gateway cert spec
	go c.handleCertificateChange(defaultCertSpec, stop)

	return nil
}

// createAndStoreGatewayCert creates a certificate for the given certificate spec and stores
// it in the referenced k8s secret if the spec is valid.
func (c *client) createAndStoreGatewayCert(spec configv1alpha2.IngressGatewayCertSpec) error {
	if len(spec.SubjectAltNames) == 0 {
		return fmt.Errorf("Ingress gateway certificate spec must specify at least 1 SAN")
	}

	// Validate the validity duration
	if _, err := time.ParseDuration(spec.ValidityDuration); err != nil {
		return fmt.Errorf("Invalid cert duration '%s' specified: %w", spec.ValidityDuration, err)
	}

	// Validate the secret ref
	if spec.Secret.Name == "" || spec.Secret.Namespace == "" {
		return fmt.Errorf("Ingress gateway cert secret's name and namespace cannot be nil, got %s/%s", spec.Secret.Namespace, spec.Secret.Name)
	}

	// Issue a certificate
	// FSM only support configuring a single SAN per cert, so pick the first one
	certCN := spec.SubjectAltNames[0]

	// A certificate for this CN may be cached already. Delete it before issuing a new certificate.
	c.certProvider.ReleaseCertificate(certCN)
	issuedCert, err := c.certProvider.IssueCertificate(certCN, certificate.IngressGateway, certificate.FullCNProvided())
	if err != nil {
		return fmt.Errorf("Error issuing a certificate for ingress gateway: %w", err)
	}

	// Store the certificate in the referenced secret
	if err := c.storeCertInSecret(issuedCert, spec.Secret); err != nil {
		return fmt.Errorf("Error storing ingress gateway cert in secret %s/%s: %w", spec.Secret.Namespace, spec.Secret.Name, err)
	}

	return nil
}

// createAndStoreAccessCert creates a certificate for the given certificate spec and stores
// it in the referenced k8s secret if the spec is valid.
func (c *client) createAndStoreAccessCert(spec policyv1alpha1.AccessCertSpec) error {
	if len(spec.SubjectAltNames) == 0 {
		return fmt.Errorf("Ingress gateway certificate spec must specify at least 1 SAN")
	}

	// Validate the secret ref
	if spec.Secret.Name == "" || spec.Secret.Namespace == "" {
		return fmt.Errorf("Access cert secret's name and namespace cannot be nil, got %s/%s", spec.Secret.Namespace, spec.Secret.Name)
	}

	// Issue a certificate
	// FSM only support configuring a single SAN per cert, so pick the first one
	certCN := spec.SubjectAltNames[0]

	// A certificate for this CN may be cached already. Delete it before issuing a new certificate.
	c.certProvider.ReleaseCertificate(certCN)
	issuedCert, err := c.certProvider.IssueCertificate(certCN, certificate.Service, certificate.FullCNProvided())
	if err != nil {
		return fmt.Errorf("Error issuing a certificate for access: %w", err)
	}

	// Store the certificate in the referenced secret
	if err := c.storeCertInSecret(issuedCert, spec.Secret); err != nil {
		return fmt.Errorf("Error storing access cert in secret %s/%s: %w", spec.Secret.Namespace, spec.Secret.Name, err)
	}

	return nil
}

// storeCertInSecret stores the certificate in the specified k8s TLS secret
func (c *client) storeCertInSecret(cert *certificate.Certificate, secret corev1.SecretReference) error {
	secretData := map[string][]byte{
		"ca.crt":  cert.GetTrustedCAs(),
		"tls.crt": cert.GetCertificateChain(),
		"tls.key": cert.GetPrivateKey(),
	}

	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: secretData,
	}

	_, err := c.kubeClient.CoreV1().Secrets(secret.Namespace).Create(context.Background(), sec, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		_, err = c.kubeClient.CoreV1().Secrets(secret.Namespace).Update(context.Background(), sec, metav1.UpdateOptions{})
	}
	return err
}

// handleCertificateChange updates the gateway certificate and secret when the MeshConfig resource changes or
// when the corresponding gateway certificate is rotated.
func (c *client) handleCertificateChange(currentCertSpec *configv1alpha2.IngressGatewayCertSpec, stop <-chan struct{}) {
	kubePubSub := c.msgBroker.GetKubeEventPubSub()
	meshConfigUpdateChan := kubePubSub.Sub(announcements.MeshConfigUpdated.String())
	defer c.msgBroker.Unsub(kubePubSub, meshConfigUpdateChan)

	accessCertChan := kubePubSub.Sub(announcements.AccessCertAdded.String(), announcements.AccessCertUpdated.String(), announcements.AccessCertDeleted.String())
	defer c.msgBroker.Unsub(kubePubSub, accessCertChan)

	certPubSub := c.msgBroker.GetCertPubSub()
	certRotateChan := certPubSub.Sub(announcements.CertificateRotated.String())
	defer c.msgBroker.Unsub(certPubSub, certRotateChan)

	accessCertCache := make(map[certificate.CommonName]*policyv1alpha1.AccessCert)

	for {
		select {
		// MeshConfig was updated
		case msg, ok := <-meshConfigUpdateChan:
			if success, newCertSpec := c.doMeshConfigUpdateChan(currentCertSpec, ok, msg); success {
				currentCertSpec = newCertSpec
			}

		// AccessCert was updated
		case msg, ok := <-accessCertChan:
			c.doAccessCertChan(ok, msg, accessCertCache)

		// A certificate was rotated
		case msg, ok := <-certRotateChan:
			c.doCertRotateChan(currentCertSpec, ok, msg, accessCertCache)

		case <-stop:
			return
		}
	}
}

func (c *client) doMeshConfigUpdateChan(currentCertSpec *configv1alpha2.IngressGatewayCertSpec, ok bool, msg interface{}) (bool, *configv1alpha2.IngressGatewayCertSpec) {
	if !ok {
		log.Warn().Msgf("Notification channel closed for MeshConfig")
		return false, nil
	}

	event, ok := msg.(events.PubSubMessage)
	if !ok {
		log.Error().Msgf("Received unexpected message %T on channel, expected PubSubMessage", event)
		return false, nil
	}

	updatedMeshConfig, ok := event.NewObj.(*configv1alpha2.MeshConfig)
	if !ok {
		log.Error().Msgf("Received unexpected object %T, expected MeshConfig", updatedMeshConfig)
		return false, nil
	}
	newCertSpec := updatedMeshConfig.Spec.Certificate.IngressGateway
	if reflect.DeepEqual(currentCertSpec, newCertSpec) {
		log.Debug().Msg("Ingress gateway certificate spec was not updated")
		return false, nil
	}
	if newCertSpec == nil && currentCertSpec != nil {
		// Implies the certificate reference was removed, delete the corresponding secret and certificate
		if err := c.removeGatewayCertAndSecret(*currentCertSpec); err != nil {
			log.Error().Err(err).Msg("Error removing stale gateway certificate/secret")
		}
	} else if newCertSpec != nil {
		// New cert spec is not nil and is not the same as the current cert spec, update required
		err := c.createAndStoreGatewayCert(*newCertSpec)
		if err != nil {
			log.Error().Err(err).Msgf("Error updating ingress gateway cert and secret")
		}
	}
	return true, newCertSpec
}

func (c *client) doAccessCertChan(ok bool, msg interface{}, accessCertCache map[certificate.CommonName]*policyv1alpha1.AccessCert) {
	if !ok {
		log.Warn().Msg("Notification channel closed for AccessCert")
		return
	}

	event, ok := msg.(events.PubSubMessage)
	if !ok {
		log.Error().Msgf("Received unexpected message %T on channel, expected PubSubMessage", event)
		return
	}

	newAccessCert, newOk := event.NewObj.(*policyv1alpha1.AccessCert)
	if !c.checkAccessCertPermission(newOk, newAccessCert) {
		log.Warn().Msg("FSM is prohibited to issue certificates for external services.")
		return
	}

	oldAccessCert, oldOk := event.OldObj.(*policyv1alpha1.AccessCert)
	if oldOk && oldAccessCert != nil {
		delete(accessCertCache, certificate.CommonName(oldAccessCert.Spec.SubjectAltNames[0]))
		if err := c.removeAccessCert(oldAccessCert, newOk, newAccessCert); err != nil {
			return
		}
	}

	if newOk && newAccessCert != nil {
		c.issueAccessCert(newAccessCert, accessCertCache)
	}
}

func (c *client) doCertRotateChan(currentCertSpec *configv1alpha2.IngressGatewayCertSpec, ok bool, msg interface{}, accessCertCache map[certificate.CommonName]*policyv1alpha1.AccessCert) {
	if !ok {
		log.Warn().Msg("Notification channel closed for certificate rotation")
		return
	}

	event, ok := msg.(events.PubSubMessage)
	if !ok {
		log.Error().Msgf("Received unexpected message %T on channel, expected PubSubMessage", event)
		return
	}
	cert, ok := event.NewObj.(*certificate.Certificate)
	if !ok {
		log.Error().Msgf("Received unexpected message %T on cert rotation channel, expected Certificate", cert)
		return
	}

	if currentCertSpec != nil && cert.GetCommonName() == certificate.CommonName(currentCertSpec.SubjectAltNames[0]) {
		log.Info().Msg("Ingress gateway certificate was rotated, updating corresponding secret")
		if err := c.createAndStoreGatewayCert(*currentCertSpec); err != nil {
			log.Error().Err(err).Msgf("Error updating ingress gateway cert secret after cert rotation")
		}
		return
	}

	if accessCertSpec, exist := accessCertCache[cert.GetCommonName()]; exist {
		log.Info().Msg("Access certificate was rotated, updating corresponding secret")
		if err := c.createAndStoreAccessCert(accessCertSpec.Spec); err != nil {
			log.Error().Err(err).Msgf("Error updating access cert secret after cert rotation")
		}
	}
}

func (c *client) checkAccessCertPermission(newOk bool, newAccessCert *policyv1alpha1.AccessCert) bool {
	if newOk && newAccessCert != nil && !c.cfg.GetFeatureFlags().EnableAccessCertPolicy {
		newAccessCert.Status = policyv1alpha1.AccessCertStatus{
			CurrentStatus: "error",
			Reason:        "FSM is prohibited to issue certificates for external services.",
		}
		if _, err := c.kubeController.UpdateStatus(newAccessCert); err != nil {
			log.Error().Err(err).Msg("Error updating status for AccessCert")
		}
		return false
	}
	return true
}

func (c *client) issueAccessCert(newAccessCert *policyv1alpha1.AccessCert, accessCertCache map[certificate.CommonName]*policyv1alpha1.AccessCert) {
	if err := c.createAndStoreAccessCert(newAccessCert.Spec); err != nil {
		log.Error().Err(err).Msgf("Error creating new access cert")
		newAccessCert.Status = policyv1alpha1.AccessCertStatus{
			CurrentStatus: "error",
			Reason:        err.Error(),
		}
		if _, err := c.kubeController.UpdateStatus(newAccessCert); err != nil {
			log.Error().Err(err).Msg("Error updating status for AccessCert")
		}
	} else {
		accessCertCache[certificate.CommonName(newAccessCert.Spec.SubjectAltNames[0])] = newAccessCert
		newAccessCert.Status = policyv1alpha1.AccessCertStatus{
			CurrentStatus: "committed",
			Reason:        "successfully committed by the system",
		}
		if _, err := c.kubeController.UpdateStatus(newAccessCert); err != nil {
			log.Error().Err(err).Msg("Error updating status for AccessCert")
		}
	}
}

func (c *client) removeAccessCert(oldAccessCert *policyv1alpha1.AccessCert, newOk bool, newAccessCert *policyv1alpha1.AccessCert) error {
	err := c.removeAccessCertAndSecret(oldAccessCert.Spec)
	if err != nil {
		log.Error().Err(err).Msgf("Error deleting old access cert")
		if newOk && newAccessCert != nil {
			newAccessCert.Status = policyv1alpha1.AccessCertStatus{
				CurrentStatus: "error",
				Reason:        err.Error(),
			}
			if _, statusErr := c.kubeController.UpdateStatus(newAccessCert); statusErr != nil {
				log.Error().Err(statusErr).Msg("Error updating status for AccessCert")
			}
		}
	}
	return err
}

// removeGatewayCertAndSecret removes the secret and certificate corresponding to the existing cert spec
func (c *client) removeGatewayCertAndSecret(storedCertSpec configv1alpha2.IngressGatewayCertSpec) error {
	err := c.kubeClient.CoreV1().Secrets(storedCertSpec.Secret.Namespace).Delete(context.Background(), storedCertSpec.Secret.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	c.certProvider.ReleaseCertificate(storedCertSpec.SubjectAltNames[0]) // Only single SAN is supported in certs

	return nil
}

// removeAccessCertAndSecret removes the secret and certificate corresponding to the existing cert spec
func (c *client) removeAccessCertAndSecret(storedCertSpec policyv1alpha1.AccessCertSpec) error {
	err := c.kubeClient.CoreV1().Secrets(storedCertSpec.Secret.Namespace).Delete(context.Background(), storedCertSpec.Secret.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	c.certProvider.ReleaseCertificate(storedCertSpec.SubjectAltNames[0]) // Only single SAN is supported in certs

	return nil
}
