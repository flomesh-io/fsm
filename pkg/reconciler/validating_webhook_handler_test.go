package reconciler

import (
	"context"
	"strconv"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/metricsstore"
	"github.com/flomesh-io/fsm/pkg/validator"
)

func TestValidatingWebhookEventHandlerUpdateFunc(t *testing.T) {
	testCases := []struct {
		name         string
		originalVwhc admissionv1.ValidatingWebhookConfiguration
		updatedVwhc  admissionv1.ValidatingWebhookConfiguration
		vwhcUpdated  bool
	}{
		{
			name: "webhook name and namespace selector changed",
			originalVwhc: admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.FSMControllerName,
						constants.FSMAppVersionLabelKey:  fsmVersion,
						constants.FSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.ValidatingWebhook{
					{
						Name: validator.ValidatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.FSMKubeResourceMonitorAnnotation: meshName,
							},
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      constants.IgnoreLabel,
									Operator: metav1.LabelSelectorOpDoesNotExist,
								},
							},
						},
					},
				},
			},
			updatedVwhc: admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.FSMControllerName,
						constants.FSMAppVersionLabelKey:  fsmVersion,
						constants.FSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.ValidatingWebhook{
					{
						Name: validator.ValidatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name-new",
								Path:      &testWebhookServicePath,
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.FSMKubeResourceMonitorAnnotation: meshName,
								"some": "label",
							},
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      constants.IgnoreLabel,
									Operator: metav1.LabelSelectorOpDoesNotExist,
								},
							},
						},
					},
				},
			},
			vwhcUpdated: true,
		},
		{
			name: "validating webhook new label added",
			originalVwhc: admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.FSMControllerName,
						constants.FSMAppVersionLabelKey:  fsmVersion,
						constants.FSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.ValidatingWebhook{
					{
						Name: validator.ValidatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
					},
				},
			},
			updatedVwhc: admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.FSMControllerName,
						constants.FSMAppVersionLabelKey:  fsmVersion,
						constants.FSMAppInstanceLabelKey: meshName,
						"some":                           "label",
					},
				},
				Webhooks: []admissionv1.ValidatingWebhook{
					{
						Name: validator.ValidatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
					},
				},
			},
			vwhcUpdated: false,
		},
		{
			name: "validating webhook name changed",
			originalVwhc: admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.FSMControllerName,
						constants.FSMAppVersionLabelKey:  fsmVersion,
						constants.FSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.ValidatingWebhook{
					{
						Name: validator.ValidatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
					},
				},
			},
			updatedVwhc: admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName-updated--",
					Labels: map[string]string{
						constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.FSMControllerName,
						constants.FSMAppVersionLabelKey:  fsmVersion,
						constants.FSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.ValidatingWebhook{
					{
						Name: validator.ValidatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name-new",
								Path:      &testWebhookServicePath,
							},
						},
					},
				},
			},
			vwhcUpdated: true,
		},
	}

	metricsstore.DefaultMetricsStore.Start(metricsstore.DefaultMetricsStore.ReconciliationTotal)
	defer metricsstore.DefaultMetricsStore.Stop(metricsstore.DefaultMetricsStore.ReconciliationTotal)
	expectedMetric := 0

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)

			kubeClient := testclient.NewSimpleClientset(&tc.originalVwhc)

			c := client{
				kubeClient:      kubeClient,
				meshName:        meshName,
				fsmVersion:      fsmVersion,
				apiServerClient: nil,
				informers:       informerCollection{},
			}
			// Invoke update handler
			handlers := c.validatingWebhookEventHandler()
			handlers.UpdateFunc(&tc.originalVwhc, &tc.updatedVwhc)

			if tc.vwhcUpdated {
				a.Equal(&tc.originalVwhc, &tc.updatedVwhc)
			} else {
				a.NotEqual(&tc.originalVwhc, &tc.updatedVwhc)
			}

			vwhc, err := c.kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), tc.originalVwhc.Name, metav1.GetOptions{})
			a.Nil(err)

			if tc.vwhcUpdated {
				a.Equal(vwhc, &tc.updatedVwhc)
			} else {
				a.Equal(vwhc, &tc.originalVwhc)
			}

			if tc.vwhcUpdated {
				expectedMetric++
			}
			if expectedMetric > 0 {
				a.True(metricsstore.DefaultMetricsStore.Contains(`fsm_reconciliation_total{kind="ValidatingWebhookConfiguration"} ` + strconv.Itoa(expectedMetric) + "\n"))
			}
		})
	}
}

func TestValidatingWebhookEventHandlerDeleteFunc(t *testing.T) {
	originalVwhc := admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "--webhookName--",
			Labels: map[string]string{
				constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
				constants.ReconcileLabel:         strconv.FormatBool(true),
				constants.AppLabel:               constants.FSMControllerName,
				constants.FSMAppVersionLabelKey:  fsmVersion,
				constants.FSMAppInstanceLabelKey: meshName,
			},
		},
		Webhooks: []admissionv1.ValidatingWebhook{
			{
				Name: validator.ValidatingWebhookName,
				ClientConfig: v1.WebhookClientConfig{
					Service: &v1.ServiceReference{
						Namespace: "test-namespace",
						Name:      "test-service-name",
						Path:      &testWebhookServicePath,
					},
				},
			},
		},
	}

	a := tassert.New(t)
	kubeClient := testclient.NewSimpleClientset()

	c := client{
		kubeClient:      kubeClient,
		meshName:        meshName,
		fsmVersion:      fsmVersion,
		apiServerClient: nil,
		informers:       informerCollection{},
	}
	// Invoke delete handler
	handlers := c.validatingWebhookEventHandler()
	handlers.DeleteFunc(&originalVwhc)

	// verify mwhc exists after deletion
	mwhc, err := c.kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), originalVwhc.Name, metav1.GetOptions{})
	a.Nil(err)
	a.Equal(mwhc, &originalVwhc)
}

func TestIsValidatingWebhookUpdated(t *testing.T) {
	testCases := []struct {
		name         string
		originalVwhc admissionv1.ValidatingWebhookConfiguration
		updatedVwhc  admissionv1.ValidatingWebhookConfiguration
		vwhcUpdated  bool
	}{
		{
			name: "webhook name and namespace selector changed",
			originalVwhc: admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.FSMControllerName,
						constants.FSMAppVersionLabelKey:  fsmVersion,
						constants.FSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.ValidatingWebhook{
					{
						Name: validator.ValidatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.FSMKubeResourceMonitorAnnotation: meshName,
							},
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      constants.IgnoreLabel,
									Operator: metav1.LabelSelectorOpDoesNotExist,
								},
							},
						},
					},
				},
			},
			updatedVwhc: admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.FSMControllerName,
						constants.FSMAppVersionLabelKey:  fsmVersion,
						constants.FSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.ValidatingWebhook{
					{
						Name: validator.ValidatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name-new",
								Path:      &testWebhookServicePath,
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.FSMKubeResourceMonitorAnnotation: meshName,
								"some": "label",
							},
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      constants.IgnoreLabel,
									Operator: metav1.LabelSelectorOpDoesNotExist,
								},
							},
						},
					},
				},
			},
			vwhcUpdated: true,
		},
		{
			name: "validating webhook new label added",
			originalVwhc: admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.FSMControllerName,
						constants.FSMAppVersionLabelKey:  fsmVersion,
						constants.FSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.ValidatingWebhook{
					{
						Name: validator.ValidatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
					},
				},
			},
			updatedVwhc: admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.FSMControllerName,
						constants.FSMAppVersionLabelKey:  fsmVersion,
						constants.FSMAppInstanceLabelKey: meshName,
						"some":                           "label",
					},
				},
				Webhooks: []admissionv1.ValidatingWebhook{
					{
						Name: validator.ValidatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
					},
				},
			},
			vwhcUpdated: false,
		},
		{
			name: "validating webhook name changed",
			originalVwhc: admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.FSMControllerName,
						constants.FSMAppVersionLabelKey:  fsmVersion,
						constants.FSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.ValidatingWebhook{
					{
						Name: validator.ValidatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
					},
				},
			},
			updatedVwhc: admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName-updated--",
					Labels: map[string]string{
						constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.FSMControllerName,
						constants.FSMAppVersionLabelKey:  fsmVersion,
						constants.FSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.ValidatingWebhook{
					{
						Name: validator.ValidatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name-new",
								Path:      &testWebhookServicePath,
							},
						},
					},
				},
			},
			vwhcUpdated: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			c := client{
				fsmVersion: fsmVersion,
			}
			result := c.isValidatingWebhookUpdated(&tc.originalVwhc, &tc.updatedVwhc)
			assert.Equal(result, tc.vwhcUpdated)
		})
	}
}
