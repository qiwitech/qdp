package client

import (
	"context"
	"math/rand"
	"strconv"

	"github.com/pkg/errors"
	cli "gopkg.in/urfave/cli.v2"

	"github.com/qiwitech/qdp/proto/apipb"
)

var (
	MetaKey string
)

func Transfer(cx *cli.Context) error {
	args := cx.Args()

	u, err := accountFromArgs(args)
	if err != nil {
		cli.ShowSubcommandHelp(cx)
		return err
	}

	req := &apipb.TransferRequest{Sender: u}

	if err := parseTransferItems(req, args.Slice()[1:]); err != nil {
		cli.ShowSubcommandHelp(cx)
		return err
	}

	if MetaKey != "" {
		key, err := MetaKeyDecode(MetaKey)
		if err != nil {
			return errors.Wrap(err, "meta key decode")
		}
		phases := [...]string{"new_moon", "first_quarter", "full_moon", "third_quarter"}
		req.Metadata = &apipb.Meta{
			Key: key,
			Index: map[string][]byte{
				"moon": []byte(phases[rand.Intn(4)]),
			},
		}
	}

	if err := connect(); err != nil {
		return err
	}

	h, err := getPrevHash(req.Sender)
	if err != nil {
		return errors.Wrap(err, "prev hash")
	}
	req.PrevHash = h

	sett, err := api.GetLastSettings(context.TODO(), &apipb.GetLastSettingsRequest{Account: u})
	if err != nil {
		return err
	}
	err = inspectStatus(sett.Status)
	if err != nil {
		return err
	}

	req.SettingsId = sett.Id

	for i := 0; i < RepeatFlag; i++ {
		err = SignTransfer(req)
		if err != nil {
			return errors.Wrap(err, "sign")
		}

		resp, err := api.ProcessTransfer(context.TODO(), req)
		if err != nil {
			return err
		}

		err = inspectStatus(resp.Status)
		if err != nil {
			return err
		}

		printResponse(cx, resp)

		req.PrevHash = resp.Hash
	}

	return nil
}

func PrevHash(cx *cli.Context) error {
	args := cx.Args()

	u, err := accountFromArgs(args)
	if err != nil {
		cli.ShowSubcommandHelp(cx)
		return err
	}

	if err := connect(); err != nil {
		return err
	}

	resp, err := api.GetPrevHash(context.TODO(), &apipb.GetPrevHashRequest{Account: u})
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

func Balance(cx *cli.Context) error {
	args := cx.Args()

	u, err := accountFromArgs(args)
	if err != nil {
		cli.ShowSubcommandHelp(cx)
		return err
	}

	if err := connect(); err != nil {
		return err
	}

	resp, err := api.GetBalance(context.TODO(), &apipb.GetBalanceRequest{Account: u})
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

func getPrevHash(u uint64) (string, error) {
	req := &apipb.GetPrevHashRequest{Account: u}

	resp, err := api.GetPrevHash(context.TODO(), req)
	if err != nil {
		return "", err
	}
	err = inspectStatus(resp.Status)
	if err != nil {
		return "", err
	}

	return resp.Hash, nil
}

func parseTransferItems(req *apipb.TransferRequest, a []string) error {
	if len(a) < 2 || len(a)%2 != 0 {
		return ErrArguments
	}
	base := 10
	if HexFlag {
		base = 16
	}

	for i := 0; i < len(a); i += 2 {
		u, err := strconv.ParseUint(a[i], base, 64)
		if err != nil {
			return err
		}
		am, err := strconv.ParseInt(a[i+1], 10, 64)
		if err != nil {
			return err
		}

		req.Batch = append(req.Batch, &apipb.TransferItem{Receiver: u, Amount: am})
	}

	return nil
}
