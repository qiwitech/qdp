package client

import (
	"github.com/qiwitech/graceful"
	"github.com/qiwitech/qdp/proto/apipb"
)

var (
	api apipb.APIServiceInterface

	AddrFlag string
)

var codec graceful.Codec = &graceful.JSONCodec{}

func connect() error {
	if api != nil {
		return nil
	}

	client, err := graceful.NewClient("http://"+AddrFlag+"/v1/", codec, nil)
	if err != nil {
		return err
	}

	api = apipb.NewAPIServiceHTTPClient(client)

	return nil
}
