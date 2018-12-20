package api

import (
	"context"
	"errors"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/mocks"
	"github.com/qiwitech/qdp/proto/gatepb"
)

func TestRouterGetClientForAccount(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	router := mocks.NewMockRouter(mock)

	var returnedClient struct {
		gatepb.ProcessorServiceInterface
	}
	r := NewRouter(router, func(url string) gatepb.ProcessorServiceInterface {
		return returnedClient
	})

	router.EXPECT().GetHostByKey("1").Return("")

	cl, err := r.getClientForAccount(1)
	assert.EqualError(t, err, "no gate url for account 1")
	assert.Nil(t, cl)

	router.EXPECT().GetHostByKey("1").Return("host-a")

	cl, err = r.getClientForAccount(1)
	assert.NoError(t, err)
	assert.True(t, returnedClient == cl)
}

func TestRouterCheckReroute(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	router := mocks.NewMockRouter(mock)

	r := NewRouter(router, func(url string) gatepb.ProcessorServiceInterface {
		return nil
	})

	_, err := r.checkReroute(nil)
	assert.EqualError(t, err, "no status")

	ok, err := r.checkReroute(&gatepb.Status{Code: gatepb.TransferCode_SEE_OTHER})

	assert.EqualError(t, err, "no routing info")
	assert.False(t, ok)

	// --
	nodes := []string{"node_a", "node_b", "node_c"}
	router.EXPECT().SetNodes(gomock.Any()).Do(func(got []string) {
		assert.Equal(t, nodes, got)
	})

	rt := &gatepb.RouteMap{Nodes: nodes, Target: "target"}
	m, _ := proto.Marshal(rt)
	details := []*any.Any{{Value: m, TypeUrl: proto.MessageName(rt)}}

	ok, err = r.checkReroute(&gatepb.Status{Code: gatepb.TransferCode_SEE_OTHER, Details: details})

	assert.NoError(t, err)
	assert.True(t, ok)

	func() {
		details[0].TypeUrl = "qew"
		defer func() {
			details[0].TypeUrl = proto.MessageName(rt)
		}()

		assert.Panics(t, func() {
			r.checkReroute(&gatepb.Status{Code: gatepb.TransferCode_SEE_OTHER, Details: details})
		})
	}()

	func() {
		details[0].TypeUrl = proto.MessageName(&gatepb.TestRouteMapAnotherType{})
		defer func() {
			details[0].TypeUrl = proto.MessageName(rt)
		}()

		ok, err = r.checkReroute(&gatepb.Status{Code: gatepb.TransferCode_SEE_OTHER, Details: details})

		assert.EqualError(t, err, "api reroute: invalid route map type: *gatepb.TestRouteMapAnotherType")
		assert.False(t, ok)
	}()

	// --
	details[0].Value = details[0].Value[:2]
	ok, err = r.checkReroute(&gatepb.Status{Code: gatepb.TransferCode_SEE_OTHER, Details: details})

	assert.EqualError(t, err, "api reroute: decode status details: unexpected EOF")
	assert.False(t, ok)

	// --
	details[0].Value = nil
	ok, err = r.checkReroute(&gatepb.Status{Code: gatepb.TransferCode_SEE_OTHER, Details: details})

	assert.EqualError(t, err, "api reroute: empty nodes list")
	assert.False(t, ok)

	// --
	ok, err = r.checkReroute(&gatepb.Status{Code: gatepb.TransferCode_OK})

	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestRouterRouteAndRetry(t *testing.T) {
	r := NewRouter(nil, nil)

	respErr := errors.New("some error")
	r.getClient = func(uint64) (gatepb.ProcessorServiceInterface, error) {
		return nil, respErr
	}
	r.checkReroute = func(s *gatepb.Status) (bool, error) {
		return false, respErr
	}

	err := r.routeAndRetry(200, func(cl gatepb.ProcessorServiceInterface) (*gatepb.Status, error) {
		return nil, nil
	})
	assert.EqualError(t, err, "api route: "+respErr.Error())

	// --
	r.getClient = func(uint64) (gatepb.ProcessorServiceInterface, error) {
		return nil, nil
	}

	err = r.routeAndRetry(200, func(cl gatepb.ProcessorServiceInterface) (*gatepb.Status, error) {
		return nil, nil
	})
	assert.EqualError(t, err, "api reroute check: "+respErr.Error())

	// --
	r.checkReroute = func(s *gatepb.Status) (bool, error) {
		return false, nil
	}

	err = r.routeAndRetry(200, func(cl gatepb.ProcessorServiceInterface) (*gatepb.Status, error) {
		return nil, nil
	})
	assert.NoError(t, err)

	// --
	reroute := true
	r.checkReroute = func(s *gatepb.Status) (bool, error) {
		if reroute {
			reroute = false
			return true, nil
		}
		return false, nil
	}

	err = r.routeAndRetry(200, func(cl gatepb.ProcessorServiceInterface) (*gatepb.Status, error) {
		return nil, nil
	})
	assert.NoError(t, err)

	// --
	err = r.routeAndRetry(200, func(cl gatepb.ProcessorServiceInterface) (*gatepb.Status, error) {
		return nil, respErr
	})
	assert.True(t, err == respErr)
}

func TestRouterProcessTransfer(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	router := mocks.NewMockRouter(mock)

	callReq := &gatepb.TransferRequest{Sender: 501}
	callRes := &gatepb.TransferResponse{Status: &gatepb.Status{}}

	r := NewRouter(router, func(url string) gatepb.ProcessorServiceInterface {
		return nil
	})
	r.routeCall = func(acc uint64, f func(gatepb.ProcessorServiceInterface) (*gatepb.Status, error)) error {
		assert.Equal(t, uint64(501), acc)

		cl := mocks.NewMockProcessorServiceInterface(mock)
		cl.EXPECT().ProcessTransfer(gomock.Any(), gomock.Any()).Return(callRes, nil).Do(func(ctx context.Context, req *gatepb.TransferRequest) {
			assert.True(t, context.TODO() == ctx)
			assert.True(t, callReq == req)
		})

		_, err := f(cl)

		return err
	}

	res, err := r.ProcessTransfer(context.TODO(), callReq)
	assert.NoError(t, err)
	assert.True(t, callRes == res)
}

func TestRouterGetPrevHash(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	router := mocks.NewMockRouter(mock)

	callReq := &gatepb.GetPrevHashRequest{Account: 501}
	callRes := &gatepb.GetPrevHashResponse{Status: &gatepb.Status{}}

	r := NewRouter(router, func(url string) gatepb.ProcessorServiceInterface {
		return nil
	})
	r.routeCall = func(acc uint64, f func(gatepb.ProcessorServiceInterface) (*gatepb.Status, error)) error {
		assert.Equal(t, uint64(501), acc)

		cl := mocks.NewMockProcessorServiceInterface(mock)
		cl.EXPECT().GetPrevHash(gomock.Any(), gomock.Any()).Return(callRes, nil).Do(func(ctx context.Context, req *gatepb.GetPrevHashRequest) {
			assert.True(t, context.TODO() == ctx)
			assert.True(t, callReq == req)
		})

		_, err := f(cl)

		return err
	}

	res, err := r.GetPrevHash(context.TODO(), callReq)
	assert.NoError(t, err)
	assert.True(t, callRes == res)
}

func TestRouterGetBalance(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	router := mocks.NewMockRouter(mock)

	callReq := &gatepb.GetBalanceRequest{Account: 501}
	callRes := &gatepb.GetBalanceResponse{Status: &gatepb.Status{}}

	r := NewRouter(router, func(url string) gatepb.ProcessorServiceInterface {
		return nil
	})
	r.routeCall = func(acc uint64, f func(gatepb.ProcessorServiceInterface) (*gatepb.Status, error)) error {
		assert.Equal(t, uint64(501), acc)

		cl := mocks.NewMockProcessorServiceInterface(mock)
		cl.EXPECT().GetBalance(gomock.Any(), gomock.Any()).Return(callRes, nil).Do(func(ctx context.Context, req *gatepb.GetBalanceRequest) {
			assert.True(t, context.TODO() == ctx)
			assert.True(t, callReq == req)
		})

		_, err := f(cl)

		return err
	}

	res, err := r.GetBalance(context.TODO(), callReq)
	assert.NoError(t, err)
	assert.True(t, callRes == res)
}

func TestRouterUpdateSettings(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	router := mocks.NewMockRouter(mock)

	callReq := &gatepb.SettingsRequest{Account: 501}
	callRes := &gatepb.SettingsResponse{Status: &gatepb.Status{}}

	r := NewRouter(router, func(url string) gatepb.ProcessorServiceInterface {
		return nil
	})
	r.routeCall = func(acc uint64, f func(gatepb.ProcessorServiceInterface) (*gatepb.Status, error)) error {
		assert.Equal(t, uint64(501), acc)

		cl := mocks.NewMockProcessorServiceInterface(mock)
		cl.EXPECT().UpdateSettings(gomock.Any(), gomock.Any()).Return(callRes, nil).Do(func(ctx context.Context, req *gatepb.SettingsRequest) {
			assert.True(t, context.TODO() == ctx)
			assert.True(t, callReq == req)
		})

		_, err := f(cl)

		return err
	}

	res, err := r.UpdateSettings(context.TODO(), callReq)
	assert.NoError(t, err)
	assert.True(t, callRes == res)
}

func TestRouterGetLastSettings(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	router := mocks.NewMockRouter(mock)

	callReq := &gatepb.GetLastSettingsRequest{Account: 501}
	callRes := &gatepb.GetLastSettingsResponse{Status: &gatepb.Status{}}

	r := NewRouter(router, func(url string) gatepb.ProcessorServiceInterface {
		return nil
	})
	r.routeCall = func(acc uint64, f func(gatepb.ProcessorServiceInterface) (*gatepb.Status, error)) error {
		assert.Equal(t, uint64(501), acc)

		cl := mocks.NewMockProcessorServiceInterface(mock)
		cl.EXPECT().GetLastSettings(gomock.Any(), gomock.Any()).Return(callRes, nil).Do(func(ctx context.Context, req *gatepb.GetLastSettingsRequest) {
			assert.True(t, context.TODO() == ctx)
			assert.True(t, callReq == req)
		})

		_, err := f(cl)

		return err
	}

	res, err := r.GetLastSettings(context.TODO(), callReq)
	assert.NoError(t, err)
	assert.True(t, callRes == res)
}
