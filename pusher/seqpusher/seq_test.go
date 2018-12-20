package seqpusher

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/mocks"
	"github.com/qiwitech/qdp/pt"
)

func TestSeqpusherOK(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	var ss []pt.Pusher
	for i := 0; i < 3; i++ {
		s := mocks.NewMockPusher(mock)
		s.EXPECT().Push(gomock.Any(), gomock.Any()).Times(1).Return(nil)
		ss = append(ss, s)
	}

	m := New(ss...)

	err := m.Push(context.TODO(), nil)
	assert.NoError(t, err)
}

func TestSeqpusherError(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("test err")

	var ss []pt.Pusher
	for i := 0; i < 3; i++ {
		s := mocks.NewMockPusher(mock)
		if i != 2 {
			s.EXPECT().Push(gomock.Any(), gomock.Any()).Times(1).Return(nil)
		} else {
			s.EXPECT().Push(gomock.Any(), gomock.Any()).Times(1).Return(testErr)
		}
		ss = append(ss, s)
	}

	m := New(ss...)

	err := m.Push(context.TODO(), nil)
	assert.EqualError(t, err, "seqpusher: "+testErr.Error())
}
