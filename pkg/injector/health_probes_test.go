package injector

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestGetPort(t *testing.T) {
	tests := []struct {
		name           string
		port           intstr.IntOrString
		containerPorts *[]corev1.ContainerPort
		expectedPort   int32
		expectedErr    error
	}{
		{
			name:           "no container ports",
			port:           intstr.FromString("-some-port-"),
			containerPorts: &[]corev1.ContainerPort{},
			expectedErr:    errNoMatchingPort,
		},
		{
			name: "named port",
			port: intstr.FromString("-some-port-"),
			containerPorts: &[]corev1.ContainerPort{
				{Name: "-some-port-", ContainerPort: 2344},
				{Name: "-some-other-port-", ContainerPort: 8877},
			},
			expectedPort: 2344,
		},
		{
			name: "numbered port",
			port: intstr.FromInt(9955),
			containerPorts: &[]corev1.ContainerPort{
				{Name: "-some-port-", ContainerPort: 2344},
				{Name: "-some-other-port-", ContainerPort: 8877},
			},
			expectedPort: 9955,
		},
		{
			name: "no matching named ports",
			port: intstr.FromString("-another-port-"),
			containerPorts: &[]corev1.ContainerPort{
				{Name: "-some-port-", ContainerPort: 2344},
				{Name: "-some-other-port-", ContainerPort: 8877},
			},
			expectedErr: errNoMatchingPort,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			actual, err := getPort(test.port, test.containerPorts)

			if test.expectedErr != nil {
				assert.ErrorIs(err, errNoMatchingPort)
				return
			}

			assert.Nil(err)
			assert.Equal(test.expectedPort, actual)
		})
	}
}
