#!/usr/bin/env bash

set -eu

if [ -n "${1}" ]; then
	dir=$1
else
	echo "usage: $0 <dir>";
	exit;
fi

cd $dir

protoc -I/usr/local/include -I${GOPATH}/src/github.com/gogo/protobuf/protobuf -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway -I. -I${GOPATH}/src --gogo_out=. --httpgw_out=. --tcprpc_out=. *.proto
