package validator

import (
	"context"
	"strconv"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/constants"
)

var (
	ingressRule = admissionregv1.RuleWithOperations{
		Operations: []admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		Rule: admissionregv1.Rule{
			APIGroups:   []string{"policy.flomesh.io"},
			APIVersions: []string{"v1alpha1"},
			Resources:   []string{"ingressbackends", "egresses", "egressgateways"},
		},
	}

	pluginRule = admissionregv1.RuleWithOperations{
		Operations: []admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		Rule: admissionregv1.Rule{
			APIGroups:   []string{"plugin.flomesh.io"},
			APIVersions: []string{"v1alpha1"},
			Resources:   []string{"plugins", "pluginchains", "pluginconfigs"},
		},
	}

	trafficTargetRule = admissionregv1.RuleWithOperations{
		Operations: []admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		Rule: admissionregv1.Rule{
			APIGroups:   []string{"access.smi-spec.io"},
			APIVersions: []string{"v1alpha3"},
			Resources:   []string{"traffictargets"},
		},
	}
)

func TestCreateValidatingWebhook(t *testing.T) {
	webhookPath := validationAPIPath
	webhookPort := int32(constants.ValidatorWebhookPort)
	fsmVersion := "test-version"
	webhookName := "--webhookName--"
	meshName := "test-mesh"
	fsmNamespace := "test-namespace"
	enableReconciler := true

	testCases := []struct {
		name                  string
		validateTrafficTarget bool
		expectedRules         []admissionregv1.RuleWithOperations
	}{
		{
			name:                  "with smi validation enabled",
			validateTrafficTarget: true,
			expectedRules:         []admissionregv1.RuleWithOperations{ingressRule, pluginRule, trafficTargetRule},
		},
		{
			name:                  "with smi validation disabled",
			validateTrafficTarget: false,
			expectedRules:         []admissionregv1.RuleWithOperations{ingressRule, pluginRule},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			cert := &certificate.Certificate{}

			kubeClient := fake.NewSimpleClientset()

			err := createOrUpdateValidatingWebhook(kubeClient, cert, webhookName, meshName, fsmNamespace, fsmVersion, tc.validateTrafficTarget, enableReconciler)
			assert.Nil(err)
			webhooks, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(context.TODO(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Len(webhooks.Items, 1)

			wh := webhooks.Items[0]
			assert.Len(wh.Webhooks, 1)
			assert.Equal(wh.ObjectMeta.Name, webhookName)
			assert.EqualValues(wh.ObjectMeta.Labels, map[string]string{
				constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
				constants.FSMAppInstanceLabelKey: meshName,
				constants.FSMAppVersionLabelKey:  fsmVersion,
				constants.AppLabel:               constants.FSMControllerName,
				constants.ReconcileLabel:         strconv.FormatBool(true),
			})
			assert.Equal(wh.Webhooks[0].ClientConfig.Service.Namespace, fsmNamespace)
			assert.Equal(wh.Webhooks[0].ClientConfig.Service.Name, ValidatorWebhookSvc)
			assert.Equal(wh.Webhooks[0].ClientConfig.Service.Path, &webhookPath)
			assert.Equal(wh.Webhooks[0].ClientConfig.Service.Port, &webhookPort)

			assert.Equal(wh.Webhooks[0].NamespaceSelector.MatchLabels[constants.FSMKubeResourceMonitorAnnotation], meshName)
			assert.EqualValues(wh.Webhooks[0].NamespaceSelector.MatchExpressions, []metav1.LabelSelectorRequirement{
				{
					Key:      constants.IgnoreLabel,
					Operator: metav1.LabelSelectorOpDoesNotExist,
				},
				{
					Key:      "name",
					Operator: metav1.LabelSelectorOpNotIn,
					Values:   []string{fsmNamespace},
				},
				{
					Key:      "control-plane",
					Operator: metav1.LabelSelectorOpDoesNotExist,
				},
			})

			assert.ElementsMatch(wh.Webhooks[0].Rules, tc.expectedRules)
			assert.Equal(wh.Webhooks[0].AdmissionReviewVersions, []string{"v1"})
		})
	}
}
