#!/bin/bash

MAX_NODES=25

if test -z "$1"
then
  echo "ERROR: you must pass value for number of nodes you wish to init, e.g.: 10"
  exit 1
fi


if [ "$1" -gt "$MAX_NODES" ];
then
  printf "If you wish to run with a value larger that 25, you must edit the Dockerfile in the go-filecoin repo\n
  Where to edit:\n
  ENV GENSETUP_COUNT 25 #<--SET THIS VALUE\n

  After edit you must rebuild the docker file:\n
  $ docker build -t go-filecoin .
  "
  exit 1
fi

# create a testbed for the iptb nodes
iptb testbed create --count "$1" --type dockerfilecoin --force

printf "Initalizing %d nodes\n" "$1"
iptb init -- --genesisfile=/data/genesis.car

printf "Starting %d nodes\n" "$1"
iptb start -- --block-time=5s

printf "Configuring %d nodes\n" "$1"
for i in $(eval echo {0..$1})
do
  minerAddr=$(iptb run "$i" cat /data/minerAddr$i | tail -n 2 | head -n 1)
  iptb run "$i" -- go-filecoin config mining.minerAddress \"\\\"$minerAddr\\\"\"
  iptb run "$i" -- go-filecoin wallet import /data/walletKey$i
done

printf "Connecting %d nodes\n" "$1"
iptb connect

printf "Complete! %d nodes connected and ready to mine >.>" "$1"
