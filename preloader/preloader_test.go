package preloader

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/mocks"
	"github.com/qiwitech/qdp/pt"
)

func TestPreloader(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	bc := mocks.NewMockBigChain(mock)
	c := mocks.NewMockChain(mock)
	sc := mocks.NewMockSettingsChain(mock)

	p := New(c, sc, bc)

	testCallback = make(chan error)

	txns := []pt.Txn{
		{Sender: 1, Receiver: 2},
		{Sender: 4, Receiver: 5},
	}
	sett := &pt.Settings{Account: 4}

	bc.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(txns, sett, nil)
	c.EXPECT().PutTo(pt.AccID(0), gomock.Any()).Times(1).Do(func(acc pt.AccID, ts []pt.Txn) {
		assert.Equal(t, txns, ts)
	})
	sc.EXPECT().Put(gomock.Any()).Times(1).Do(func(s *pt.Settings) {
		assert.Equal(t, sett, s)
	})

	done := make(chan error, 1)
	go func() {
		err := <-testCallback
		done <- err
	}()

	err := p.Preload(context.TODO(), 0)
	assert.NoError(t, err)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Errorf("callback does not called")
	}

	err = p.Preload(context.TODO(), 0)
	assert.NoError(t, err)
}

func TestPreloaderErr(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	bc := mocks.NewMockBigChain(mock)
	c := mocks.NewMockChain(mock)
	sc := mocks.NewMockSettingsChain(mock)

	p := New(c, sc, bc)

	testCallback = make(chan error)

	respErr := errors.New("test error")

	bc.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil, respErr)
	done := make(chan error, 1)
	go func() {
		err := <-testCallback
		done <- err
	}()
	err := p.Preload(context.TODO(), 4)
	assert.EqualError(t, errors.Cause(err), respErr.Error())

	select {
	case err := <-done:
		assert.EqualError(t, err, "chain preloader: "+respErr.Error())
	case <-time.After(time.Second):
		t.Errorf("callback does not called")
	}

	txns := []pt.Txn{
		{Sender: 1, Receiver: 2},
		{Sender: 4, Receiver: 5},
	}
	sett := &pt.Settings{Account: 4}

	bc.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(txns, sett, nil)
	c.EXPECT().PutTo(pt.AccID(4), gomock.Any()).Times(1).Do(func(acc pt.AccID, ts []pt.Txn) {
		assert.Equal(t, txns, ts)
	})
	sc.EXPECT().Put(gomock.Any()).Times(1).Do(func(s *pt.Settings) {
		assert.Equal(t, sett, s)
	})

	done = make(chan error, 1)
	go func() {
		err := <-testCallback
		done <- err
	}()

	err = p.Preload(context.TODO(), 4)
	assert.NoError(t, err)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Errorf("callback does not called")
	}
}

func TestPreloader2Accounts(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	bc := mocks.NewMockBigChain(mock)
	c := mocks.NewMockChain(mock)
	sc := mocks.NewMockSettingsChain(mock)

	p := New(c, sc, bc)

	testCallback = make(chan error)

	// first account
	txns := []pt.Txn{
		{Sender: 10, Receiver: 2},
		{Sender: 40, Receiver: 5},
	}
	sett := &pt.Settings{Account: 10}

	bc.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(txns, sett, nil)
	c.EXPECT().PutTo(pt.AccID(10), gomock.Any()).Times(1).Do(func(acc pt.AccID, ts []pt.Txn) {
		assert.Equal(t, txns, ts)
	})
	sc.EXPECT().Put(gomock.Any()).Times(1).Do(func(s *pt.Settings) {
		assert.Equal(t, sett, s)
	})

	done := make(chan error, 1)
	go func() {
		err := <-testCallback
		done <- err
	}()

	err := p.Preload(context.TODO(), 10)
	assert.NoError(t, err)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Errorf("callback does not called")
	}

	// second account
	txns = []pt.Txn{
		{Sender: 1, Receiver: 20},
		{Sender: 4, Receiver: 50},
	}
	sett = &pt.Settings{Account: 20}

	bc.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(txns, sett, nil)
	c.EXPECT().PutTo(pt.AccID(20), gomock.Any()).Times(1).Do(func(acc pt.AccID, ts []pt.Txn) {
		assert.Equal(t, txns, ts)
	})
	sc.EXPECT().Put(gomock.Any()).Times(1).Do(func(s *pt.Settings) {
		assert.Equal(t, sett, s)
	})

	done = make(chan error, 1)
	go func() {
		err := <-testCallback
		done <- err
	}()

	err = p.Preload(context.TODO(), 20)
	assert.NoError(t, err)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Errorf("callback does not called")
	}
}
