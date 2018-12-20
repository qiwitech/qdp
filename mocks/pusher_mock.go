// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/qiwitech/qdp/proto/pusherpb (interfaces: PusherServiceInterface)

package mocks

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	pusherpb "github.com/qiwitech/qdp/proto/pusherpb"
)

// Mock of PusherServiceInterface interface
type MockPusherServiceInterface struct {
	ctrl     *gomock.Controller
	recorder *_MockPusherServiceInterfaceRecorder
}

// Recorder for MockPusherServiceInterface (not exported)
type _MockPusherServiceInterfaceRecorder struct {
	mock *MockPusherServiceInterface
}

func NewMockPusherServiceInterface(ctrl *gomock.Controller) *MockPusherServiceInterface {
	mock := &MockPusherServiceInterface{ctrl: ctrl}
	mock.recorder = &_MockPusherServiceInterfaceRecorder{mock}
	return mock
}

func (_m *MockPusherServiceInterface) EXPECT() *_MockPusherServiceInterfaceRecorder {
	return _m.recorder
}

func (_m *MockPusherServiceInterface) Push(_param0 context.Context, _param1 *pusherpb.PushRequest) (*pusherpb.PushResponse, error) {
	ret := _m.ctrl.Call(_m, "Push", _param0, _param1)
	ret0, _ := ret[0].(*pusherpb.PushResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockPusherServiceInterfaceRecorder) Push(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Push", arg0, arg1)
}
