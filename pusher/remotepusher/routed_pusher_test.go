package remotepusher

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/mocks"
	"github.com/qiwitech/qdp/pt"
)

func TestRoutedPusher(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	r := mocks.NewMockRouter(mock)
	rp := NewRoutedPusher(r)

	rp.client = func(baseurl string) pt.Pusher {
		ph := mocks.NewMockPusher(mock)
		ph.EXPECT().Push(gomock.Any(), gomock.Any()).Return(nil)
		return ph
	}

	r.EXPECT().GetHostByKey(gomock.Any()).Return("host")
	r.EXPECT().IsSelf(gomock.Any()).Return(true)

	err := rp.Push(context.TODO(), []pt.Txn{{ID: 1, Sender: 0, Receiver: 10, Amount: 10, Balance: 100}})
	assert.NoError(t, err)

	r.EXPECT().GetHostByKey(gomock.Any()).Return("")

	err = rp.Push(context.TODO(), []pt.Txn{{ID: 1, Sender: 0, Receiver: 10, Amount: 10, Balance: 100}})
	assert.EqualError(t, err, "routed push failed: no remote push url for account 10")

	r.EXPECT().GetHostByKey(gomock.Any()).Return("localhost")
	r.EXPECT().IsSelf(gomock.Any()).Return(false)

	err = rp.Push(context.TODO(), []pt.Txn{{ID: 1, Sender: 0, Receiver: 10, Amount: 10, Balance: 100}})
	assert.NoError(t, err)
}
