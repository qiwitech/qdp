package client

import (
	"errors"
	"strconv"

	cli "gopkg.in/urfave/cli.v2"
)

var (
	ErrArguments      = errors.New("arguments error")
	ErrNotImplemented = errors.New("not implemented yet")
)

func accountFromArgs(a cli.Args) (uint64, error) {
	if a.Len() < 1 {
		return 0, ErrArguments
	}
	base := 10
	if HexFlag {
		base = 16
	}
	u, err := strconv.ParseUint(a.First(), base, 64)
	return u, err
}
