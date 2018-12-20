package client

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"
	cli "gopkg.in/urfave/cli.v2"

	"github.com/qiwitech/qdp/proto/apipb"
)

var (
	MetaEncodingFlag string
)

func GetByMetaKey(cx *cli.Context) error {
	args := cx.Args()

	req := &apipb.GetByMetaKeyRequest{
		Keys: make([][]byte, args.Len()),
	}

	for i := range args.Slice() {
		if d, err := MetaKeyDecode(args.Get(i)); err != nil {
			return errors.Wrap(err, "decode meta key")
		} else {
			req.Keys[i] = d
		}
	}

	if err := connect(); err != nil {
		return err
	}

	resp, err := api.GetByMetaKey(context.TODO(), req)
	if err != nil {
		return err
	}

	err = inspectStatus(resp.Status)
	if err != nil {
		return err
	}

	printResponse(cx, resp)

	return nil
}

var metaKeyDecode func(string) ([]byte, error)

func MetaKeyDecode(s string) ([]byte, error) {
	if metaKeyDecode == nil {
		switch MetaEncodingFlag {
		case "base64", "b64":
			metaKeyDecode = base64.RawStdEncoding.DecodeString
		case "raw":
			metaKeyDecode = func(s string) ([]byte, error) { return []byte(s), nil }
		default:
			return nil, fmt.Errorf("unsupported meta key encoding: %v", MetaEncodingFlag)
		}
	}

	return metaKeyDecode(s)
}
