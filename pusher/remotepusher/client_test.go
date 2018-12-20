package remotepusher

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/mocks"
	"github.com/qiwitech/qdp/proto/pusherpb"
	"github.com/qiwitech/qdp/pt"
)

func TestEmptyPush(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	p := mocks.NewMockPusherServiceInterface(mock)
	cl := NewClient(p, nil)

	err := cl.Push(context.TODO(), []pt.Txn{})
	assert.NoError(t, err)
}

func TestFailedPush(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	p := mocks.NewMockPusherServiceInterface(mock)
	cl := NewClient(p, nil)

	p.EXPECT().Push(gomock.Any(), gomock.Any()).Return(&pusherpb.PushResponse{Status: &pusherpb.Status{Code: 0}}, nil)
	err := cl.Push(context.TODO(), []pt.Txn{{}})
	assert.NoError(t, err)

	p.EXPECT().Push(gomock.Any(), gomock.Any()).Return(nil, errors.New("fake error"))
	err = cl.Push(context.TODO(), []pt.Txn{{}})
	assert.EqualError(t, err, "remote push failed: fake error")

	p.EXPECT().Push(gomock.Any(), gomock.Any()).Return(nil, nil)
	err = cl.Push(context.TODO(), []pt.Txn{{}})
	assert.EqualError(t, err, "remote push failed: empty response")

	p.EXPECT().Push(gomock.Any(), gomock.Any()).Return(&pusherpb.PushResponse{
		Status: &pusherpb.Status{
			Code:    int32(pusherpb.PushCode_INTERNAL_ERROR),
			Message: "some internal error",
		},
	}, nil)
	err = cl.Push(context.TODO(), []pt.Txn{{}})
	assert.EqualError(t, err, "remote push failed: some internal error")

}

func TestHTTPClient(t *testing.T) {
	cl := NewHTTPClient("localhost")
	assert.NotNil(t, cl)
}
