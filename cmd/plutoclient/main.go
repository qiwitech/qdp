package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	cli "gopkg.in/urfave/cli.v2"

	"github.com/qiwitech/qdp/client"
)

var (
	Version = "HEAD"
	Commit  = "dev"
)

func main() {
	app := &cli.App{}
	app.Name = os.Args[0]
	app.Usage = "console plutos client"
	app.Version = Version + "-" + Commit
	app.EnableShellCompletion = true
	app.Commands = []*cli.Command{
		{
			Name:        "transfer",
			Usage:       "<sender> {<receiver> <amount>}",
			Description: "make transfer",
			Action:      client.Transfer,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "metakey", Aliases: []string{"m"}, Destination: &client.MetaKey},
			},
		},
		{
			// TODO(nik): name it lasthash?
			Name:        "prevhash",
			Usage:       "<account>",
			Description: "loads prev_hash for account",
			Action:      client.PrevHash,
		},
		{
			// TODO(nik): name it lasthash?
			Name:        "balance",
			Usage:       "<account>",
			Description: "loads account balance",
			Action:      client.Balance,
		},
		{
			Name:  "settings",
			Usage: "Perform settings operations",
			Subcommands: []*cli.Command{
				{
					Name:   "last",
					Usage:  "<account> - show current account settings",
					Action: client.GetLastSettings,
				},
				{
					Name:        "keygen",
					Usage:       "<account> - change account keys",
					Description: "change public key on settings",
					Action:      client.Keygen,
				},
				{
					Name:        "verify",
					Usage:       "<account> <true|yes|t|y|1 or false|no|f|n|0> - update settings VerifyTransferSign field",
					Description: "change VerifyTransferSign field on settings",
					Action:      client.UpdateVerifyTransferSign,
				},
				{
					Name:        "datahash",
					Usage:       "<account> <hash> - update settings DataHash field",
					Description: "change DataHash field on settings",
					Action:      client.UpdateDataHash,
				},
			},
			Action: client.GetLastSettings,
		},
		{
			Name:        "history",
			Usage:       "<account>",
			Description: "loads history for account",
			Action:      client.GetHistory,
			Flags: []cli.Flag{
				&cli.IntFlag{Name: "limit", Aliases: []string{"l"}, Value: 10},
				&cli.StringFlag{Name: "token", Aliases: []string{"t"}},
			},
		},
		{
			Name:        "meta",
			Usage:       "{<key>}",
			Description: "retrieves transaction by meta key",
			Action:      client.GetByMetaKey,
		},
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "keysdb", Aliases: []string{"d"}, Value: ".plutoclientdb", Destination: &client.KeysDBFlag},
		&cli.StringFlag{Name: "addr", Aliases: []string{"a"}, Value: ":9090", Destination: &client.AddrFlag, EnvVars: []string{"PLUTOAPI"}},
		&cli.BoolFlag{Name: "verbose", Value: false, Destination: &client.VerboseFlag},
		&cli.BoolFlag{Name: "hex", Value: false, Destination: &client.HexFlag},
		&cli.IntFlag{Name: "repeat", Aliases: []string{"r"}, Value: 1, Destination: &client.RepeatFlag},
		&cli.StringFlag{Name: "meta-encoding", Value: "raw", Destination: &client.MetaEncodingFlag},
	}

	rand.Seed(time.Now().UnixNano())

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
}
