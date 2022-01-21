package main

import (
	"context"
	"fmt"
	"os"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
)

var debug bool

var log = logging.Logger("main")

func home(home, path string) string {
	return fmt.Sprintf("%s/%s", home, path)
}

type Miner struct {
	owner  string
	worker string
	id     string

	h string
}

func NewMiner(owner, worker, id string) Miner {
	if owner == "" {
		owner = os.Getenv("OWNER_ADDR")
	}

	h, err := homedir.Dir()
	if err != nil {
		log.Infof("getting home directory failed: %s", err)
	}

	return Miner{
		owner:  owner,
		worker: worker,
		id:     id,
		h:      h,
	}
}

func (s Miner) MinerPath() string {
	prefix := os.Getenv("LOTUS_MINER_PATH_PREFIX")
	return home(s.h, fmt.Sprintf("%s%s", prefix, s.worker))
}

func (s Miner) MinerPathEnv() string {
	minerpath := s.MinerPath()
	return fmt.Sprintf("LOTUS_MINER_PATH=%s", minerpath)
}

func main() {
	local := []*cli.Command{
		fixAddrCmd,
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
		miner := NewMiner("", c.Args().Get(1), c.Args().Get(0))

		return miner.fixCounterMetadata(context.Background())
	},
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

	counter, err := mds.Get(datastore.NewKey(modules.StorageCounterDSPrefix))
	if err != nil {
		return err
	}

	fmt.Println(counter)

	return nil
}

var fixAddrCmd = &cli.Command{
	Name:  "fixaddr",
	Usage: "fixaddr <minerID> <minerAddress>",
	Action: func(c *cli.Context) error {
		miner := NewMiner("", c.Args().Get(1), c.Args().Get(0))

		return miner.fixMinerMetadata(context.Background())
	},
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
		return nil, fmt.Errorf("repo at '%s' is not initialized, run 'lotus-miner init' to set it up", s.MinerPath())
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

func (s Miner) fixMinerMetadata(ctx context.Context) error {
	lr, err := s.GetDatastore(ctx)
	if err != nil {
		return err
	}

	mds, err := lr.Datastore(ctx, "/metadata")
	if err != nil {
		return err
	}

	addr, err := address.NewFromString(s.id)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	if err := mds.Put(datastore.NewKey("miner-address"), addr.Bytes()); err != nil {
		return err
	}

	return nil
}

// // LotusClient returns a JSONRPC client for the Lotus API
// func LotusClient(ctx context.Context) (lotusapi.FullNode, jsonrpc.ClientCloser, error) {
// 	authToken := os.Getenv("LOTUS_TOKEN")
// 	headers := http.Header{"Authorization": []string{"Bearer " + authToken}}
// 	addr := os.Getenv("LOTUS_API")

// 	return client.NewFullNodeRPCV1(ctx, "ws://"+addr+"/rpc/v1", headers)
// }

// func LotusMinerClient(ctx context.Context) (lotusapi.StorageMiner, jsonrpc.ClientCloser, error) {
// 	authToken := os.Getenv("LOTUSMINER_TOKEN")
// 	headers := http.Header{"Authorization": []string{"Bearer " + authToken}}
// 	addr := os.Getenv("LOTUSMINER_API")
// 	if addr == "" {
// 		addr = "127.0.0.1:2345"
// 	}

// 	return client.NewStorageMinerRPCV0(ctx, "ws://"+addr+"/rpc/v0", headers)
// }

// func GetMinerAddress(ctx context.Context) (address.Address, error) {
// 	miner, closer, err := LotusMinerClient(ctx)
// 	if err != nil {
// 		return address.Address{}, err
// 	}
// 	defer closer()

// 	maddr, err := miner.ActorAddress(ctx)
// 	if err != nil {
// 		return address.Address{}, err
// 	}

// 	return maddr, nil
// }

// func (s *Service) SetMinerToken(ctx context.Context) error {
// 	content, err := ioutil.ReadFile(fmt.Sprintf("%s/token", s.MinerPath()))
// 	if err != nil {
// 		log.Infof("reading token failed: %s", err)
// 		return err
// 	}
// 	os.Setenv("LOTUSMINER_TOKEN", string(content))
// 	return nil
// }
