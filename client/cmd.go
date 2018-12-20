package client

import (
	"encoding/json"
	"errors"
	"fmt"

	cli "gopkg.in/urfave/cli.v2"

	"github.com/qiwitech/qdp/proto/apipb"
)

var (
	VerboseFlag bool
	HexFlag     bool
	RepeatFlag  int = 1
)

func inspectStatus(s *apipb.Status) error {
	if s == nil {
		return errors.New("no status")
	}

	switch s.Code {
	case 0:
		return nil
	}

	return errors.New(s.Message)
}

func printRequest(cx *cli.Context, req interface{}) {
	// TODO(nik): print to console according to flags
	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		fmt.Fprintf(cx.App.Writer, "print error: %v\ndata: %v\n", err, req)
		return
	}
	fmt.Fprintf(cx.App.Writer, "%s\n", data)
}

func printResponse(cx *cli.Context, resp interface{}) {
	// TODO(nik): print to console according to flags
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		fmt.Fprintf(cx.App.Writer, "print error: %v\ndata: %v\n", err, resp)
		return
	}
	fmt.Fprintf(cx.App.Writer, "%s\n", data)
}
