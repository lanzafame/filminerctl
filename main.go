package main

import (
	"context"
	"fmt"
	"os"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-storedcounter"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

var debug bool

var log = logging.Logger("main")

type Miner struct {
}

func main() {
	local := []*cli.Command{
		fixCounterCmd,
	}

	app := &cli.App{
		Name:     "filminerctl",
		Commands: local,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "enable debug mode",
				Value: false,
			},
		},
		Before: func(ctx *cli.Context) error {
			debug = ctx.Bool("debug")
			return nil
		},
	}
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}

var fixCounterCmd = &cli.Command{
	Name:  "fixcounter",
	Usage: "",
	Action: func(c *cli.Context) error {
		miner := Miner{}

		return miner.fixCounterMetadata(context.Background())
	},
}

type sidsc struct {
	sc *storedcounter.StoredCounter
}

func (s *sidsc) Next() (abi.SectorNumber, error) {
	i, err := s.sc.Next()
	return abi.SectorNumber(i), err
}

func (s Miner) fixCounterMetadata(ctx context.Context) error {
	lr, err := s.GetDatastore(ctx)
	if err != nil {
		return err
	}

	mds, err := lr.Datastore(ctx, "/metadata")
	if err != nil {
		return err
	}

	counter := storedcounter.New(mds, datastore.NewKey(modules.StorageCounterDSPrefix))

	i, err := counter.Next()
	if err != nil {
		return err
	}
	fmt.Println(i)

	return nil
}

func (s Miner) GetDatastore(ctx context.Context) (repo.LockedRepo, error) {
	r, err := repo.NewFS(os.Getenv("LOTUS_MINER_PATH"))
	if err != nil {
		return nil, err
	}

	ok, err := r.Exists()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("repo is not initialized, run 'lotus-miner init' to set it up")
	}

	lr, err := r.Lock(repo.StorageMiner)
	if err != nil {
		return nil, err
	}

	return lr, nil
}

func (s Miner) getMinerMetadata(ctx context.Context) (string, error) {
	lr, err := s.GetDatastore(ctx)
	if err != nil {
		return "", err
	}

	mds, err := lr.Datastore(ctx, "/metadata")
	if err != nil {
		return "", err
	}

	addrb, err := mds.Get(datastore.NewKey("miner-address"))
	if err != nil {
		return "", err
	}

	addr, err := address.NewFromBytes(addrb)
	if err != nil {
		return "", err
	}
	return addr.String(), nil
}
