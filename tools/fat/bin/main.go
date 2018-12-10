package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"time"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/commands"
	"github.com/filecoin-project/go-filecoin/protocol/storage"
	"github.com/filecoin-project/go-filecoin/types"

	"github.com/filecoin-project/go-filecoin/tools/fat"
	"github.com/filecoin-project/go-filecoin/tools/fat/process"
)

func init() {
	logging.SetAllLoggers(4)
}

type result struct {
	deals   []int
	commits []bool
}

func main() {
	deallists := [][]int{
		{100, 20, 7, 127},
	}

	results := []result{}
	for _, deals := range deallists {
		commits := Sanity(deals)

		results = append(results, result{
			deals:   deals,
			commits: commits,
		})
	}

	for _, res := range results {
		fmt.Println(res.deals)
		fmt.Println(res.commits)
		fmt.Println()
	}
}

func Cluster() {
	ctx := context.Background()
	dir, err := ioutil.TempDir("", "ACTION")
	if err != nil {
		panic(err)
	}
	env, err := fat.NewEnvironment(dir)
	if err != nil {
		panic(err)
	}

	defer env.Teardown()

	if err := env.AddGenesisMiner(ctx); err != nil {
		panic(err)
	}

	for i := 0; i < 5; i++ {
		if err := env.AddMiner(ctx); err != nil {
			panic(err)
		}
	}
}

func storeData(client *process.Filecoin, data io.Reader) (*storage.DealResponse, *commands.ClientListAsksResult, cid.Cid) {
	dcid, err := client.ClientImport(data)
	if err != nil {
		panic(err)
	}

	asks, err := client.ClientListAsks()
	if err != nil {
		client.DumpLastOutput()
		panic(err)
	}

	// Find ask created from AddAsk
	ask, err := asks.Next()
	if err != nil {
		panic(err)
	}

	// Propose the deal
	deal, err := client.ProposeStorageDeal(dcid, ask.Miner, ask.ID, 5)
	if err != nil {
		client.DumpLastOutput()
		panic(err)
	}

	return deal, ask, dcid
}

type MadeDeal struct {
	cid   cid.Cid
	miner address.Address
	data  []byte
	deal  *storage.DealResponse
}

func waitForSectorFromMiner(client *process.Filecoin, miner address.Address, after cid.Cid) (cid.Cid, error) {
	owner, err := client.GetOwner(miner)
	if err != nil {
		return cid.Undef, err
	}

	for i := 0; i < 10; i++ {
		// ? Wait around till it's sealed?
		ichain, err := client.ChainLs()
		if err != nil {
			client.DumpLastOutput()
			return cid.Undef, err
		}

		done := false
	walkchain:
		for {
			tipset, err := ichain.Next()
			if err == io.EOF {
				break
			}

			if err != nil {
				return cid.Undef, err
			}

			for _, block := range *tipset {
				for _, msg := range block.Messages {
					if msg.Message.Method == "commitSector" && msg.Message.From == owner {
						return block.Cid(), nil
					}
				}

				if block.Cid() == after {
					done = true
				}

			}

			if done {
				break walkchain
			}
		}

		time.Sleep(time.Second * 5)
	}

	return cid.Undef, fmt.Errorf("Could not find commitSector from miner")
}

func Sanity(chunkSizes []int) []bool {
	ctx := context.Background()
	dir, err := ioutil.TempDir("", "ACTION")
	if err != nil {
		panic(err)
	}
	env, err := fat.NewEnvironment(dir)
	if err != nil {
		panic(err)
	}

	defer env.Teardown()

	if err := env.AddGenesisMiner(ctx); err != nil {
		panic(err)
	}

	if err := env.AddMiner(ctx); err != nil {
		panic(err)
	}

	if err := env.AddNode(ctx); err != nil {
		panic(err)
	}

	if err := env.AddNode(ctx); err != nil {
		panic(err)
	}

	procs := env.Processes()

	if err := procs[1].MiningStart(); err != nil {
		procs[1].DumpLastOutput()
		panic(err)
	}

	value, _ := types.NewAttoFILFromFILString("10")
	c, err := procs[1].AddAsk(
		procs[1].MinerOwner,
		procs[1].MinerAddress,
		value,
		big.NewInt(10000),
	)
	if err != nil {
		procs[1].DumpLastOutput()
		panic(err)
	}

	_, err = procs[1].MessageWait(c)
	if err != nil {
		procs[1].DumpLastOutput()
		panic(err)
	}

	var deals []MadeDeal
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for i, size := range chunkSizes {
		data := []byte{}
		for x := 0; x < size; x++ {
			data = append(data, alphabet[(x+i)%len(alphabet)])
		}

		rdr := bytes.NewReader(data)
		deal, ask, cid := storeData(procs[2], rdr)

		deals = append(deals, MadeDeal{
			miner: ask.Miner,
			cid:   cid,
			data:  data,
			deal:  deal,
		})
	}

	time.Sleep(time.Second * 140)

	dealsCommited := []bool{}
	for _, deal := range deals {
		commited := false
		for y := 0; y < 5; y++ {
			dr, err := procs[2].QueryStorageDeal(deal.deal.Proposal)
			if err != nil {
				procs[2].DumpLastOutput()
				panic(err)
			}

			if dr.State == storage.Posted {
				commited = true
				break
			}

			time.Sleep(time.Second * 5)
		}

		dealsCommited = append(dealsCommited, commited)
		if commited {
			bits, err := procs[3].RetrievePiece(deal.miner, deal.cid)
			procs[3].DumpLastOutput()
			if err != nil {
				panic(err)
			}

			bb, err := ioutil.ReadAll(bits)
			if err != nil {
				panic(err)
			}

			fmt.Println(string(bb))
			fmt.Println(len(bb))
		}
	}

	return dealsCommited
}

/*
func transferData(miner, source, sink *process.Filecoin, data io.Reader, price types.AttoFIL, expiry *big.Int) error {
	msgid, err := miner.AddAsk(miner.MinerOwner, miner.MinerAddress, price, expiry)
	if err != nil {
		miner.DumpLastOutput()
		panic(err)
	}

	if _, err = miner.MessageWait(msgid); err != nil {
		miner.DumpLastOutput()
		panic(err)
	}

	deal, ask, cid := storeData(source, data)
}
*/
