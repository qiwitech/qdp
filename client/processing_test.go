package client

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/proto/apipb"
)

func TestTransfer(t *testing.T) {
	KeysDBFlag = ""

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	api := initMockAPI(ctrl)

	// arguments
	c, stdOut, stdErr := mockCli([]string{"11111", "22222", "10"})

	// api
	api.EXPECT().GetPrevHash(gomock.Any(), gomock.Any()).
		Times(1).
		Return(&apipb.GetPrevHashResponse{Status: &apipb.Status{}, Hash: "prev_hash"}, nil)
	api.EXPECT().GetLastSettings(gomock.Any(), gomock.Any()).
		Times(1).
		Return(&apipb.GetLastSettingsResponse{Status: &apipb.Status{}}, nil)
	api.EXPECT().ProcessTransfer(gomock.Any(), gomock.Any()).
		Times(1).
		Return(&apipb.TransferResponse{Status: &apipb.Status{}, Hash: "txn_hash"}, nil)

	// call
	err := Transfer(c)
	assert.NoError(t, err)

	assert.Equal(t, "{\n  \"status\": {},\n  \"hash\": \"txn_hash\"\n}\n", stdOut())
	assert.Equal(t, "", stdErr())
}

func TestParseTransferItems(t *testing.T) {
	testData := map[*apipb.TransferRequest][]string{
		&apipb.TransferRequest{
			Batch: []*apipb.TransferItem{
				{Receiver: 0, Amount: 111},
				{Receiver: 99999999999999, Amount: 1},
			},
		}: []string{
			"0", "111",
			"99999999999999", "1",
		},
	}

	for want, args := range testData {
		got := &apipb.TransferRequest{}
		err := parseTransferItems(got, args)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(want, got) {
			t.Errorf(`Want "%v", got "%v"`, want.Batch, got.Batch)
		}
	}
}
