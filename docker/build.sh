#!/usr/bin/env bash

set -e

repobase=rnd.im
list="plutos plutoapi sqldb"

cd $(dirname $0)

if [ ! -z "$1" ]
then
	list="$@"
fi

workdir=`mktemp -d -t plutos_build_XXXXXX`
function rmworkdir {
	rm -rf $workdir
}
trap rmworkdir 0

function buildcont {
	service=$1
	echo "Building $service..."
	go build -o $service/$service ../cmd/$service
	docker build --build-arg BINARY=$service -t $service $service
	#docker tag $service $repobase/$service
	#docker push $repobase/$service
}

for service in $list; do
	buildcont $service
done
