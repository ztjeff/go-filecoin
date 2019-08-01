#!/usr/bin/env bash
set -x

TEST_OR_LIVE="live"
WORKDIR="$(pwd)"
SCRATCHDIR="/tmp"

docker stop filecoin-0 || true
docker rm filecoin-0 || true

rm -rf $SCRATCHDIR/filecoins/0 || true
mkdir $SCRATCHDIR/filecoins/0

docker run \
-v $WORKDIR/fixtures/$TEST_OR_LIVE:/var/filecoin/car \
-v $SCRATCHDIR/filecoins/0:/var/local/filecoin \
--entrypoint=/usr/local/bin/go-filecoin \
657871693752.dkr.ecr.us-east-1.amazonaws.com/filecoin:nightly-24832-674d31 \
init \
--genesisfile=/var/filecoin/car/genesis.car \
--repodir=/var/local/filecoin/repo \
--sectordir=/var/local/filecoin/sectors \
--auto-seal-interval-seconds 300

# Init gets run as root :(
sudo chown -R $USER:$USER $SCRATCHDIR/filecoins/0

docker run -d --name filecoin-0 --hostname filecoin-0 -p 9000:9000 -p 3453:3453 -p 9400:9400 \
--entrypoint=/usr/local/bin/devnet_start \
-v $WORKDIR/fixtures/$TEST_OR_LIVE:/var/filecoin/car  \
-v $SCRATCHDIR/filecoins/0:/var/local/filecoin \
-e IPFS_LOGGING_FMT=nocolor \
-e FILECOIN_PATH=/var/local/filecoin/repo \
-e GO_FILECOIN_LOG_LEVEL=4 \
-e GO_FILECOIN_LOG_JSON=1 \
--log-driver json-file \
--log-opt max-size=10m 657871693752.dkr.ecr.us-east-1.amazonaws.com/filecoin:nightly-24832-674d31 daemon --repodir=/var/local/filecoin/repo --block-time=30s


GAS_PRICE="0.0001"
GAS_LIMIT="1000"

filecoin_exec="go-filecoin --repodir=/var/local/filecoin/repo"

minerAddr=$(cat $HOME/tmp/filecoin-daemon/car/gen.json | grep -v Fixture | jq -r '.Miners[] | select(.Owner == 0).Address')

docker exec "filecoin-0" $filecoin_exec config mining.minerAddress "${minerAddr}"

minerOwner=$(docker exec "filecoin-0" $filecoin_exec wallet import "/var/filecoin/car/0.key" | sed -e 's/^"//' -e 's/"$$//')

docker exec "filecoin-0" $filecoin_exec config wallet.defaultAddress $minerOwner

peerID=$(docker exec "filecoin-0" $filecoin_exec id | grep -v Fixture | jq ".ID" -r)

docker exec "filecoin-0" $filecoin_exec miner update-peerid --from=$minerOwner --gas-price $GAS_PRICE --gas-limit $GAS_LIMIT "$minerAddr" "$peerID"

