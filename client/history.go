package client

import (
	"context"

	cli "gopkg.in/urfave/cli.v2"

	"github.com/qiwitech/qdp/proto/apipb"
)

func GetHistory(cx *cli.Context) error {
	args := cx.Args()
	u, err := accountFromArgs(args)
	if err != nil {
		cli.ShowSubcommandHelp(cx)
		return err
	}
	lim := cx.Int("limit")
	token := cx.String("token")

	if err := connect(); err != nil {
		return err
	}

	resp, err := api.GetHistory(context.TODO(), &apipb.GetHistoryRequest{Account: u, Limit: uint32(lim), Token: token})
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
