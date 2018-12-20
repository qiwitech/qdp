package client

import (
	"bytes"
	"flag"

	"github.com/golang/mock/gomock"
	cli "gopkg.in/urfave/cli.v2"
)

func initMockAPI(ctrl *gomock.Controller) *MockAPIServiceInterface {
	mock := NewMockAPIServiceInterface(ctrl)
	api = mock
	return mock
}

func mockCli(args []string) (*cli.Context, func() string, func() string) {
	// io
	stdOut := &bytes.Buffer{}
	stdErr := &bytes.Buffer{}
	// app
	app := &cli.App{}
	app.Writer = stdOut
	app.ErrWriter = stdErr

	// arguments
	flags := flag.NewFlagSet("test", 0)
	flags.Parse(args)

	c := cli.NewContext(app, flags, nil)

	return c, stdOut.String, stdErr.String
}
