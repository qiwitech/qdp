#!/usr/bin/env bash

mkdir -p bin

GOBIN=`pwd`/bin
res=plutos.tar

go install ./cmd/bench ./cmd/plutoapi ./cmd/plutoclient ./cmd/plutos ./cmd/sqldb

echo "Creating archive..."
tar cfvz "$res" bin

echo "Done."
echo "Move $res to repo jepsen to /docker/static/ dir"
echo "If both repos are at the same location it could be done by command:"
echo "  cp $res ../jepsen/docker/static/"
