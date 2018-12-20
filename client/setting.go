package client

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	cli "gopkg.in/urfave/cli.v2"

	"github.com/qiwitech/qdp/proto/apipb"
	"github.com/qiwitech/qdp/pt"
)

func GetLastSettings(cx *cli.Context) error {
	args := cx.Args()

	u, err := accountFromArgs(args)
	if err != nil {
		cli.ShowSubcommandHelp(cx)
		return err
	}

	if err := connect(); err != nil {
		return err
	}

	resp, err := api.GetLastSettings(context.TODO(), &apipb.GetLastSettingsRequest{Account: u})
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

func Keygen(cx *cli.Context) error {
	args := cx.Args()

	u, err := accountFromArgs(args)
	if err != nil {
		cli.ShowSubcommandHelp(cx)
		return err
	}

	priv, err := btcec.NewPrivateKey(curve)
	if err != nil {
		return err
	}

	pub := priv.PubKey()
	pubb := pt.PublicKey(pub.SerializeHybrid()).String()

	if err := connect(); err != nil {
		return err
	}

	s, err := api.GetLastSettings(context.TODO(), &apipb.GetLastSettingsRequest{Account: u})
	if err != nil {
		return err
	}

	sreq := &apipb.SettingsRequest{
		Account:            u,
		PrevHash:           s.Hash,
		DataHash:           s.DataHash,
		VerifyTransferSign: s.VerifyTransferSign,
		PublicKey:          pubb, // changed field
	}

	resp, err := updateSettings(cx, sreq)
	err = inspectStatus(resp.Status)
	if err != nil {
		return err
	}

	err = SavePrivateKey(u, priv)
	if err != nil {
		fmt.Fprintf(cx.App.Writer, "Can't save key to db, but it's already written. Remember it!! %x", hex.EncodeToString(priv.Serialize()))
		return err
	}

	printResponse(cx, resp)

	return nil
}

func UpdateVerifyTransferSign(cx *cli.Context) error {
	args := cx.Args()

	if args.Len() > 2 {
		cli.ShowSubcommandHelp(cx)
		return errors.New("expected max two arguments")
	}

	u, err := accountFromArgs(args)
	if err != nil {
		cli.ShowSubcommandHelp(cx)
		return err
	}

	var val bool
	switch args.Get(1) {
	case "true", "t", "1", "y", "yes":
		val = true
	case "false", "f", "0", "n", "no":
		// already false
	default:
		cli.ShowSubcommandHelp(cx)
		return fmt.Errorf("unsupported value: %v", args.Get(1))
	}

	if err := connect(); err != nil {
		return err
	}

	s, err := api.GetLastSettings(context.TODO(), &apipb.GetLastSettingsRequest{Account: u})
	if err != nil {
		return err
	}

	if s.VerifyTransferSign == val {
		// already set to equal value
		return nil
	}

	sreq := &apipb.SettingsRequest{
		Account:            u,
		PrevHash:           s.Hash,
		DataHash:           s.DataHash,
		VerifyTransferSign: val, // changed field
		PublicKey:          s.PublicKey,
	}

	resp, err := updateSettings(cx, sreq)
	err = inspectStatus(resp.Status)
	if err != nil {
		return err
	}

	printResponse(cx, resp)

	return nil
}

func UpdateDataHash(cx *cli.Context) error {
	args := cx.Args()

	if args.Len() != 2 {
		cli.ShowSubcommandHelp(cx)
		return errors.New("expected exactly two arguments")
	}

	u, err := accountFromArgs(args)
	if err != nil {
		cli.ShowSubcommandHelp(cx)
		return err
	}

	if err := connect(); err != nil {
		return err
	}

	s, err := api.GetLastSettings(context.TODO(), &apipb.GetLastSettingsRequest{Account: u})
	if err != nil {
		return err
	}

	if s.DataHash == args.Get(1) {
		// already set to equal value
		return nil
	}

	sreq := &apipb.SettingsRequest{
		Account:            u,
		PrevHash:           s.Hash,
		DataHash:           args.Get(1), // changed field
		VerifyTransferSign: s.VerifyTransferSign,
		PublicKey:          s.PublicKey,
	}

	resp, err := updateSettings(cx, sreq)
	err = inspectStatus(resp.Status)
	if err != nil {
		return err
	}

	printResponse(cx, resp)

	return nil
}

func updateSettings(cx *cli.Context, sreq *apipb.SettingsRequest) (*apipb.SettingsResponse, error) {
	prv, err := LoadPrivateKey(sreq.Account)
	if err != nil {
		return nil, err
	}

	if prv != nil {
		hash := SettingsRequestHash(sreq)
		sign, err := pt.SignTransfer(hash, prv)
		if err != nil {
			return nil, err
		}
		sreq.Sign = sign.String()
	}

	if VerboseFlag {
		printRequest(cx, sreq)
	}

	resp, err := api.UpdateSettings(context.TODO(), sreq)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
