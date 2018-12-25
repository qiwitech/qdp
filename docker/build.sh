#!/usr/bin/env bash

set -e

# Force build target to Linux X86_64
export GOOS=linux
export GOARCH=amd64

list="plutos plutoapi sqldb"

cd $(dirname $0)

if [ ! -z "$1" ]
then
	list="$@"
fi

function buildcont {
	service=$1
	echo "Building $service..."
	go build -o $service/$service ../cmd/$service
	docker build --build-arg BINARY=$service -t $service $service
}

for service in $list; do
	buildcont $service
done
