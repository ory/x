// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ory/x/hasherx (interfaces: BCryptConfigurator)

// Package hasherx_test is a generated GoMock package.
package hasherx_test

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	hasherx "github.com/ory/x/hasherx"
)

// MockBCryptConfigurator is a mock of BCryptConfigurator interface.
type MockBCryptConfigurator struct {
	ctrl     *gomock.Controller
	recorder *MockBCryptConfiguratorMockRecorder
}

// MockBCryptConfiguratorMockRecorder is the mock recorder for MockBCryptConfigurator.
type MockBCryptConfiguratorMockRecorder struct {
	mock *MockBCryptConfigurator
}

// NewMockBCryptConfigurator creates a new mock instance.
func NewMockBCryptConfigurator(ctrl *gomock.Controller) *MockBCryptConfigurator {
	mock := &MockBCryptConfigurator{ctrl: ctrl}
	mock.recorder = &MockBCryptConfiguratorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBCryptConfigurator) EXPECT() *MockBCryptConfiguratorMockRecorder {
	return m.recorder
}

// HasherBcryptConfig mocks base method.
func (m *MockBCryptConfigurator) HasherBcryptConfig(arg0 context.Context) *hasherx.BCryptConfig {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasherBcryptConfig", arg0)
	ret0, _ := ret[0].(*hasherx.BCryptConfig)
	return ret0
}

// HasherBcryptConfig indicates an expected call of HasherBcryptConfig.
func (mr *MockBCryptConfiguratorMockRecorder) HasherBcryptConfig(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasherBcryptConfig", reflect.TypeOf((*MockBCryptConfigurator)(nil).HasherBcryptConfig), arg0)
}
