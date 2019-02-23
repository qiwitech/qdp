// Code generated by protoc-gen-httpgw
// source: api_service.proto
// DO NOT EDIT!

/*
Package apipb is a http proxy.
*/

package apipb

import (
	"context"

	"github.com/pressly/chi"
	"github.com/qiwitech/graceful"
)

func NewAPIServiceHandler(srv APIServiceInterface, c graceful.Codec) graceful.Handlerer {
	return AddAPIServiceHandlers(nil, srv, c)
}
func AddAPIServiceHandlers(mux graceful.Handlerer, srv APIServiceInterface, c graceful.Codec) graceful.Handlerer {
	if mux == nil {
		mux = chi.NewMux()
	}

	mux.Handle("/ProcessTransfer", graceful.NewHandler(
		c,
		func() interface{} { return &TransferRequest{} },
		func(ctx context.Context, args interface{}) (interface{}, error) {
			return srv.ProcessTransfer(ctx, args.(*TransferRequest))
		}))

	mux.Handle("/GetPrevHash", graceful.NewHandler(
		c,
		func() interface{} { return &GetPrevHashRequest{} },
		func(ctx context.Context, args interface{}) (interface{}, error) {
			return srv.GetPrevHash(ctx, args.(*GetPrevHashRequest))
		}))

	mux.Handle("/GetBalance", graceful.NewHandler(
		c,
		func() interface{} { return &GetBalanceRequest{} },
		func(ctx context.Context, args interface{}) (interface{}, error) {
			return srv.GetBalance(ctx, args.(*GetBalanceRequest))
		}))

	mux.Handle("/UpdateSettings", graceful.NewHandler(
		c,
		func() interface{} { return &SettingsRequest{} },
		func(ctx context.Context, args interface{}) (interface{}, error) {
			return srv.UpdateSettings(ctx, args.(*SettingsRequest))
		}))

	mux.Handle("/GetLastSettings", graceful.NewHandler(
		c,
		func() interface{} { return &GetLastSettingsRequest{} },
		func(ctx context.Context, args interface{}) (interface{}, error) {
			return srv.GetLastSettings(ctx, args.(*GetLastSettingsRequest))
		}))

	mux.Handle("/GetHistory", graceful.NewHandler(
		c,
		func() interface{} { return &GetHistoryRequest{} },
		func(ctx context.Context, args interface{}) (interface{}, error) {
			return srv.GetHistory(ctx, args.(*GetHistoryRequest))
		}))

	mux.Handle("/GetByMetaKey", graceful.NewHandler(
		c,
		func() interface{} { return &GetByMetaKeyRequest{} },
		func(ctx context.Context, args interface{}) (interface{}, error) {
			return srv.GetByMetaKey(ctx, args.(*GetByMetaKeyRequest))
		}))

	mux.Handle("/SearchMeta", graceful.NewHandler(
		c,
		func() interface{} { return &SearchMetaRequest{} },
		func(ctx context.Context, args interface{}) (interface{}, error) {
			return srv.SearchMeta(ctx, args.(*SearchMetaRequest))
		}))

	mux.Handle("/PutMeta", graceful.NewHandler(
		c,
		func() interface{} { return &PutMetaRequest{} },
		func(ctx context.Context, args interface{}) (interface{}, error) {
			return srv.PutMeta(ctx, args.(*PutMetaRequest))
		}))

	return mux
}

type APIServiceHTTPClient struct {
	*graceful.Client
}

func NewAPIServiceHTTPClient(cl *graceful.Client) APIServiceHTTPClient {
	return APIServiceHTTPClient{
		Client: cl,
	}
}

func (cl APIServiceHTTPClient) ProcessTransfer(ctx context.Context, args *TransferRequest) (*TransferResponse, error) {
	var resp TransferResponse
	err := cl.Client.Call(ctx, "ProcessTransfer", args, &resp)
	return &resp, err
}

func (cl APIServiceHTTPClient) GetPrevHash(ctx context.Context, args *GetPrevHashRequest) (*GetPrevHashResponse, error) {
	var resp GetPrevHashResponse
	err := cl.Client.Call(ctx, "GetPrevHash", args, &resp)
	return &resp, err
}

func (cl APIServiceHTTPClient) GetBalance(ctx context.Context, args *GetBalanceRequest) (*GetBalanceResponse, error) {
	var resp GetBalanceResponse
	err := cl.Client.Call(ctx, "GetBalance", args, &resp)
	return &resp, err
}

func (cl APIServiceHTTPClient) UpdateSettings(ctx context.Context, args *SettingsRequest) (*SettingsResponse, error) {
	var resp SettingsResponse
	err := cl.Client.Call(ctx, "UpdateSettings", args, &resp)
	return &resp, err
}

func (cl APIServiceHTTPClient) GetLastSettings(ctx context.Context, args *GetLastSettingsRequest) (*GetLastSettingsResponse, error) {
	var resp GetLastSettingsResponse
	err := cl.Client.Call(ctx, "GetLastSettings", args, &resp)
	return &resp, err
}

func (cl APIServiceHTTPClient) GetHistory(ctx context.Context, args *GetHistoryRequest) (*GetHistoryResponse, error) {
	var resp GetHistoryResponse
	err := cl.Client.Call(ctx, "GetHistory", args, &resp)
	return &resp, err
}

func (cl APIServiceHTTPClient) GetByMetaKey(ctx context.Context, args *GetByMetaKeyRequest) (*GetByMetaKeyResponse, error) {
	var resp GetByMetaKeyResponse
	err := cl.Client.Call(ctx, "GetByMetaKey", args, &resp)
	return &resp, err
}

func (cl APIServiceHTTPClient) SearchMeta(ctx context.Context, args *SearchMetaRequest) (*SearchMetaResponse, error) {
	var resp SearchMetaResponse
	err := cl.Client.Call(ctx, "SearchMeta", args, &resp)
	return &resp, err
}

func (cl APIServiceHTTPClient) PutMeta(ctx context.Context, args *PutMetaRequest) (*PutMetaResponse, error) {
	var resp PutMetaResponse
	err := cl.Client.Call(ctx, "PutMeta", args, &resp)
	return &resp, err
}

type APIServiceInterface interface {
	ProcessTransfer(context.Context, *TransferRequest) (*TransferResponse, error)

	GetPrevHash(context.Context, *GetPrevHashRequest) (*GetPrevHashResponse, error)

	GetBalance(context.Context, *GetBalanceRequest) (*GetBalanceResponse, error)

	UpdateSettings(context.Context, *SettingsRequest) (*SettingsResponse, error)

	GetLastSettings(context.Context, *GetLastSettingsRequest) (*GetLastSettingsResponse, error)

	GetHistory(context.Context, *GetHistoryRequest) (*GetHistoryResponse, error)

	GetByMetaKey(context.Context, *GetByMetaKeyRequest) (*GetByMetaKeyResponse, error)

	SearchMeta(context.Context, *SearchMetaRequest) (*SearchMetaResponse, error)

	PutMeta(context.Context, *PutMetaRequest) (*PutMetaResponse, error)
}
