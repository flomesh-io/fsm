// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/flomesh-io/fsm/pkg/health (interfaces: Probes)

// Package health is a generated GoMock package.
package health

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockProbes is a mock of Probes interface.
type MockProbes struct {
	ctrl     *gomock.Controller
	recorder *MockProbesMockRecorder
}

// MockProbesMockRecorder is the mock recorder for MockProbes.
type MockProbesMockRecorder struct {
	mock *MockProbes
}

// NewMockProbes creates a new mock instance.
func NewMockProbes(ctrl *gomock.Controller) *MockProbes {
	mock := &MockProbes{ctrl: ctrl}
	mock.recorder = &MockProbesMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockProbes) EXPECT() *MockProbesMockRecorder {
	return m.recorder
}

// GetID mocks base method.
func (m *MockProbes) GetID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetID")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetID indicates an expected call of GetID.
func (mr *MockProbesMockRecorder) GetID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetID", reflect.TypeOf((*MockProbes)(nil).GetID))
}

// Liveness mocks base method.
func (m *MockProbes) Liveness() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Liveness")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Liveness indicates an expected call of Liveness.
func (mr *MockProbesMockRecorder) Liveness() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Liveness", reflect.TypeOf((*MockProbes)(nil).Liveness))
}

// Readiness mocks base method.
func (m *MockProbes) Readiness() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Readiness")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Readiness indicates an expected call of Readiness.
func (mr *MockProbesMockRecorder) Readiness() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Readiness", reflect.TypeOf((*MockProbes)(nil).Readiness))
}
