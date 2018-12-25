// Code generated by protoc-gen-tcpgen
// source: pusher_service.proto
// DO NOT EDIT!

/*
Package pusherpb is a http proxy.
*/

package pusherpb

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/qiwitech/tcprpc"
)

func RegisterPusherServiceHandlers(s *tcprpc.Server, prefix string, srv PusherServiceInterface) {

	s.Handle(prefix+"Push", tcprpc.NewHandler(
		func() proto.Message { return new(PushRequest) },
		func(ctx context.Context, inp proto.Message) (proto.Message, error) {
			args := inp.(*PushRequest)
			return srv.Push(ctx, args)
		}))

}

type TCPRPCPusherServiceClient struct {
	cl   *tcprpc.Client
	pref string
}

func NewTCPRPCPusherServiceClient(cl *tcprpc.Client, pref string) TCPRPCPusherServiceClient {
	return TCPRPCPusherServiceClient{
		cl:   cl,
		pref: pref,
	}
}

func (cl TCPRPCPusherServiceClient) Push(ctx context.Context, args *PushRequest) (*PushResponse, error) {
	var resp PushResponse
	err := cl.cl.Call(ctx, cl.pref+"Push", args, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type PusherServiceInterface interface {
	Push(context.Context, *PushRequest) (*PushResponse, error)
}

func RegisterSettingsPusherServiceHandlers(s *tcprpc.Server, prefix string, srv SettingsPusherServiceInterface) {

	s.Handle(prefix+"PushSettings", tcprpc.NewHandler(
		func() proto.Message { return new(PushSettingsRequest) },
		func(ctx context.Context, inp proto.Message) (proto.Message, error) {
			args := inp.(*PushSettingsRequest)
			return srv.PushSettings(ctx, args)
		}))

}

type TCPRPCSettingsPusherServiceClient struct {
	cl   *tcprpc.Client
	pref string
}

func NewTCPRPCSettingsPusherServiceClient(cl *tcprpc.Client, pref string) TCPRPCSettingsPusherServiceClient {
	return TCPRPCSettingsPusherServiceClient{
		cl:   cl,
		pref: pref,
	}
}

func (cl TCPRPCSettingsPusherServiceClient) PushSettings(ctx context.Context, args *PushSettingsRequest) (*PushSettingsResponse, error) {
	var resp PushSettingsResponse
	err := cl.cl.Call(ctx, cl.pref+"PushSettings", args, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type SettingsPusherServiceInterface interface {
	PushSettings(context.Context, *PushSettingsRequest) (*PushSettingsResponse, error)
}