// Package types contains types used by the gateway controller
package types

import (
	"fmt"
	"strings"

	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/flomesh-io/fsm/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

var (
	log = logger.New("fsm-gateway/types")
)

// Controller is the interface for the functionality provided by the resources part of the gateway.networking.k8s.io API group
type Controller interface {
	cache.ResourceEventHandler

	// Runnable runs the backend broadcast listener
	manager.Runnable

	// LeaderElectionRunnable knows if a Runnable needs to be run in the leader election mode.
	manager.LeaderElectionRunnable
}

// PolicyMatchType is the type used to represent the rate limit policy match type
type PolicyMatchType string

const (
	// PolicyMatchTypePort is the type used to represent the rate limit policy match type port
	PolicyMatchTypePort PolicyMatchType = "port"

	// PolicyMatchTypeHostnames is the type used to represent the rate limit policy match type hostnames
	PolicyMatchTypeHostnames PolicyMatchType = "hostnames"

	// PolicyMatchTypeHTTPRoute is the type used to represent the rate limit policy match type httproute
	PolicyMatchTypeHTTPRoute PolicyMatchType = "httproute"

	// PolicyMatchTypeGRPCRoute is the type used to represent the rate limit policy match type grpcroute
	PolicyMatchTypeGRPCRoute PolicyMatchType = "grpcroute"
)

// Listener is a wrapper around the Gateway API Listener object
type Listener struct {
	gwv1.Listener
	SupportedKinds []gwv1.RouteGroupKind
}

// AllowsKind returns true if the listener allows the given kind
func (l *Listener) AllowsKind(gvk schema.GroupVersionKind) bool {
	log.Debug().Msgf("[GW-CACHE] Checking if listener allows kind %s", gvk.String())
	kind := gvk.Kind
	group := gvk.Group

	for _, allowedKind := range l.SupportedKinds {
		log.Debug().Msgf("[GW-CACHE] allowedKind={%s, %s}", *allowedKind.Group, allowedKind.Kind)
		if string(allowedKind.Kind) == kind &&
			(allowedKind.Group == nil || string(*allowedKind.Group) == group) {
			return true
		}
	}

	return false
}

// RouteContext is a wrapper around the Gateway API Route object
type RouteContext struct {
	Meta         metav1.Object
	ParentRefs   []gwv1.ParentReference
	GVK          schema.GroupVersionKind
	Generation   int64
	Hostnames    []gwv1.Hostname
	Namespace    string
	ParentStatus []gwv1.RouteParentStatus
}

// CrossNamespaceFrom is the type used to represent the from part of a cross-namespace reference
type CrossNamespaceFrom struct {
	Group     string
	Kind      string
	Namespace string
}

// CrossNamespaceTo is the type used to represent the to part of a cross-namespace reference
type CrossNamespaceTo struct {
	Group     string
	Kind      string
	Namespace string
	Name      string
}

type PolicyWrapper struct {
	Policy     client.Object
	TargetRef  gwv1alpha2.NamespacedPolicyTargetReference
	Conditions []metav1.Condition
}

// hasIndex Selector

func OneTermSelector(k string) fields.Selector {
	return &hasIndex{field: k}
}

type hasIndex struct {
	field string
}

func (t *hasIndex) Matches(ls fields.Fields) bool {
	return ls.Has(t.field)
}

func (t *hasIndex) Empty() bool {
	return false
}

func (t *hasIndex) RequiresExactMatch(field string) (value string, found bool) {
	if t.field == field {
		return "", true
	}
	return "", false
}

func (t *hasIndex) Transform(fn fields.TransformFunc) (fields.Selector, error) {
	field, value, err := fn(t.field, "")
	if err != nil {
		return nil, err
	}
	if len(field) == 0 && len(value) == 0 {
		return fields.Everything(), nil
	}
	return &hasIndex{field}, nil
}

func (t *hasIndex) Requirements() fields.Requirements {
	return []fields.Requirement{{
		Field:    t.field,
		Operator: selection.Equals,
		Value:    "",
	}}
}

func (t *hasIndex) String() string {
	return fmt.Sprintf("%v", t.field)
}

func (t *hasIndex) DeepCopySelector() fields.Selector {
	if t == nil {
		return nil
	}
	out := new(hasIndex)
	*out = *t
	return out
}

func OrSelectors(selectors ...fields.Selector) fields.Selector {
	return orTerm(selectors)
}

type orTerm []fields.Selector

func (t orTerm) Matches(ls fields.Fields) bool {
	for _, q := range t {
		if q.Matches(ls) {
			return true
		}
	}

	return false
}

func (t orTerm) Empty() bool {
	if t == nil {
		return true
	}
	if len([]fields.Selector(t)) == 0 {
		return true
	}
	for i := range t {
		if !t[i].Empty() {
			return false
		}
	}
	return true
}

func (t orTerm) RequiresExactMatch(field string) (string, bool) {
	if t == nil || len([]fields.Selector(t)) == 0 {
		return "", false
	}
	for i := range t {
		if value, found := t[i].RequiresExactMatch(field); found {
			return value, found
		}
	}
	return "", false
}

func (t orTerm) Transform(fn fields.TransformFunc) (fields.Selector, error) {
	next := make([]fields.Selector, 0, len([]fields.Selector(t)))
	for _, s := range []fields.Selector(t) {
		n, err := s.Transform(fn)
		if err != nil {
			return nil, err
		}
		if !n.Empty() {
			next = append(next, n)
		}
	}
	return orTerm(next), nil
}

func (t orTerm) Requirements() fields.Requirements {
	reqs := make([]fields.Requirement, 0, len(t))
	for _, s := range []fields.Selector(t) {
		rs := s.Requirements()
		reqs = append(reqs, rs...)
	}
	return reqs
}

func (t orTerm) String() string {
	var terms []string
	for _, q := range t {
		terms = append(terms, q.String())
	}
	return strings.Join(terms, ",")
}

func (t orTerm) DeepCopySelector() fields.Selector {
	if t == nil {
		return nil
	}
	out := make([]fields.Selector, len(t))
	for i := range t {
		out[i] = t[i].DeepCopySelector()
	}
	return orTerm(out)
}
