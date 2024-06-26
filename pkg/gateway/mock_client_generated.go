// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/flomesh-io/fsm/pkg/gateway/types (interfaces: Controller)

// Package gateway is a generated GoMock package.
package gateway

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockController is a mock of Controller interface.
type MockController struct {
	ctrl     *gomock.Controller
	recorder *MockControllerMockRecorder
}

// MockControllerMockRecorder is the mock recorder for MockController.
type MockControllerMockRecorder struct {
	mock *MockController
}

// NewMockController creates a new mock instance.
func NewMockController(ctrl *gomock.Controller) *MockController {
	mock := &MockController{ctrl: ctrl}
	mock.recorder = &MockControllerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockController) EXPECT() *MockControllerMockRecorder {
	return m.recorder
}

// NeedLeaderElection mocks base method.
func (m *MockController) NeedLeaderElection() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NeedLeaderElection")
	ret0, _ := ret[0].(bool)
	return ret0
}

// NeedLeaderElection indicates an expected call of NeedLeaderElection.
func (mr *MockControllerMockRecorder) NeedLeaderElection() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NeedLeaderElection", reflect.TypeOf((*MockController)(nil).NeedLeaderElection))
}

// OnAdd mocks base method.
func (m *MockController) OnAdd(arg0 interface{}, arg1 bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnAdd", arg0, arg1)
}

// OnAdd indicates an expected call of OnAdd.
func (mr *MockControllerMockRecorder) OnAdd(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnAdd", reflect.TypeOf((*MockController)(nil).OnAdd), arg0, arg1)
}

// OnDelete mocks base method.
func (m *MockController) OnDelete(arg0 interface{}) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnDelete", arg0)
}

// OnDelete indicates an expected call of OnDelete.
func (mr *MockControllerMockRecorder) OnDelete(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnDelete", reflect.TypeOf((*MockController)(nil).OnDelete), arg0)
}

// OnUpdate mocks base method.
func (m *MockController) OnUpdate(arg0, arg1 interface{}) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnUpdate", arg0, arg1)
}

// OnUpdate indicates an expected call of OnUpdate.
func (mr *MockControllerMockRecorder) OnUpdate(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnUpdate", reflect.TypeOf((*MockController)(nil).OnUpdate), arg0, arg1)
}

// Start mocks base method.
func (m *MockController) Start(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockControllerMockRecorder) Start(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockController)(nil).Start), arg0)
}
