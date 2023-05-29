package k8s

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/flomesh-io/fsm/pkg/constants"
)

func TestGetFSMInjectorPods(t *testing.T) {
	testNamespace := "fsm-namespace"

	tests := []struct {
		testName         string
		pods             []*corev1.Pod
		expectedPodNames []string
	}{
		{
			testName: "multiple pods (fsm-injector pods and other pods) in multiple namespaces",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fsm-injector-pod-1",
						Namespace: testNamespace,
						Labels: map[string]string{
							constants.AppLabel: constants.FSMInjectorName,
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fsm-injector-pod-2",
						Namespace: testNamespace,
						Labels: map[string]string{
							constants.AppLabel: constants.FSMInjectorName,
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-other-fsm-injector-pod",
						Namespace: "some-other-namespace",
						Labels: map[string]string{
							constants.AppLabel: constants.FSMInjectorName,
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "application-pod-1",
						Namespace: testNamespace,
						Labels: map[string]string{
							constants.AppLabel: "myapp",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "application-pod-2",
						Namespace: "some-other-namespace",
						Labels: map[string]string{
							constants.AppLabel: "myapp",
						},
					},
				},
			},
			expectedPodNames: []string{
				"fsm-injector-pod-1",
				"fsm-injector-pod-2",
			},
		},
		{
			testName:         "no pods",
			pods:             []*corev1.Pod{},
			expectedPodNames: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			assert := tassert.New(t)

			objs := make([]runtime.Object, len(test.pods))
			for i := range test.pods {
				objs[i] = test.pods[i]
			}

			fakeClientSet := fake.NewSimpleClientset(objs...)
			podList := GetFSMInjectorPods(fakeClientSet, testNamespace)
			actualPodNames := make([]string, len(podList.Items))
			for i, pod := range podList.Items {
				actualPodNames[i] = pod.Name
			}
			assert.ElementsMatch(test.expectedPodNames, actualPodNames)
		})
	}
}
