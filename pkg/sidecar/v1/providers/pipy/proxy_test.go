package pipy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	tassert "github.com/stretchr/testify/assert"

	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/models"
	"github.com/flomesh-io/fsm/pkg/tests"
)

var _ = Describe("Test proxy methods", func() {
	proxyUUID := uuid.New()
	podUID := uuid.New().String()
	proxy := NewProxy(models.KindSidecar, proxyUUID, identity.New("svc-acc", "namespace"), false, tests.NewMockAddress("1.2.3.4"))

	It("creates a valid proxy", func() {
		Expect(proxy).ToNot((BeNil()))
	})

	Context("test GetConnectedAt()", func() {
		It("returns correct values", func() {
			actual := proxy.GetConnectedAt()
			Expect(actual).ToNot(Equal(uint64(0)))
		})
	})

	Context("test GetIP()", func() {
		It("returns correct values", func() {
			actual := proxy.GetIP()
			Expect(actual.Network()).To(Equal("mockNetwork"))
			Expect(actual.String()).To(Equal("1.2.3.4"))
		})
	})

	Context("test HasMetadata()", func() {
		It("returns correct values", func() {
			actual := proxy.HasMetadata()
			Expect(actual).To(BeFalse())
		})
	})

	Context("test UUID", func() {
		It("returns correct values", func() {
			Expect(proxy.UUID).To(Equal(proxyUUID))
		})
	})

	Context("test StatsHeaders()", func() {
		It("returns correct values", func() {
			actual := proxy.StatsHeaders()
			expected := map[string]string{
				"fsm-stats-namespace": "unknown",
				"fsm-stats-kind":      "unknown",
				"fsm-stats-name":      "unknown",
				"fsm-stats-pod":       "unknown",
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("test correctness proxy object creation", func() {
		It("returns correct values", func() {
			Expect(proxy.HasMetadata()).To(BeFalse())

			proxy.Metadata = &ProxyMetadata{
				UID: podUID,
			}

			Expect(proxy.HasMetadata()).To(BeTrue())
			Expect(proxy.Metadata.UID).To(Equal(podUID))
			Expect(strings.Contains(proxy.String(), fmt.Sprintf("[ProxyUUID=%s]", proxyUUID))).To(BeTrue())
		})
	})
})

func TestStatsHeaders(t *testing.T) {
	const unknown = "unknown"
	tests := []struct {
		name     string
		proxy    Proxy
		expected map[string]string
	}{
		{
			name: "nil metadata",
			proxy: Proxy{
				Metadata: nil,
			},
			expected: map[string]string{
				"fsm-stats-kind":      unknown,
				"fsm-stats-name":      unknown,
				"fsm-stats-namespace": unknown,
				"fsm-stats-pod":       unknown,
			},
		},
		{
			name: "empty metadata",
			proxy: Proxy{
				Metadata: &ProxyMetadata{},
			},
			expected: map[string]string{
				"fsm-stats-kind":      unknown,
				"fsm-stats-name":      unknown,
				"fsm-stats-namespace": unknown,
				"fsm-stats-pod":       unknown,
			},
		},
		{
			name: "full metadata",
			proxy: Proxy{
				Metadata: &ProxyMetadata{
					Name:         "pod",
					Namespace:    "ns",
					WorkloadKind: "kind",
					WorkloadName: "name",
				},
			},
			expected: map[string]string{
				"fsm-stats-kind":      "kind",
				"fsm-stats-name":      "name",
				"fsm-stats-namespace": "ns",
				"fsm-stats-pod":       "pod",
			},
		},
		{
			name: "replicaset with expected name format",
			proxy: Proxy{
				Metadata: &ProxyMetadata{
					WorkloadKind: "ReplicaSet",
					WorkloadName: "some-name-randomchars",
				},
			},
			expected: map[string]string{
				"fsm-stats-kind":      "Deployment",
				"fsm-stats-name":      "some-name",
				"fsm-stats-namespace": unknown,
				"fsm-stats-pod":       unknown,
			},
		},
		{
			name: "replicaset without expected name format",
			proxy: Proxy{
				Metadata: &ProxyMetadata{
					WorkloadKind: "ReplicaSet",
					WorkloadName: "name",
				},
			},
			expected: map[string]string{
				"fsm-stats-kind":      "ReplicaSet",
				"fsm-stats-name":      "name",
				"fsm-stats-namespace": unknown,
				"fsm-stats-pod":       unknown,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.proxy.StatsHeaders()
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestPodMetadataString(t *testing.T) {
	testCases := []struct {
		name     string
		proxy    *Proxy
		expected string
	}{
		{
			name: "with valid pod metadata",
			proxy: &Proxy{
				Metadata: &ProxyMetadata{
					UID:            "some-UID",
					Namespace:      "some-ns",
					Name:           "some-pod",
					ServiceAccount: identity.K8sServiceAccount{Name: "some-service-account"},
				},
			},
			expected: "UID=some-UID, Namespace=some-ns, Name=some-pod, ServiceAccount=some-service-account",
		},
		{
			name: "no pod metadata",
			proxy: &Proxy{
				Metadata: nil,
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := tc.proxy.MetadataString()
			assert.Equal(tc.expected, actual)
		})
	}
}
