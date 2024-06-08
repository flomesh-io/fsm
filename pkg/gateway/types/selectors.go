package types

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/selection"
)

// OneTermSelector Selector

// OneTermSelector is a selector that matches fields that have a specific index name
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

// OrSelectors Selector represents a logical OR of multiple selectors
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
