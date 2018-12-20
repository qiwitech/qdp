#!/usr/bin/env bash

protoc --proto_path=proto/apipb \
	-I. -I${GOPATH}/src/github.com/google/protobuf/src/ -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	--swagger_out=logtostderr=true:docs/ proto/apipb/api_service.proto

cat <(cat docs/api_service.swagger.json | jq '.definitions |=
	walk(
		if type == "object" and (.format | . == "uint64" or . == "int64") then
			.type |= "integer"
		else
			.
		end) |
	walk(
		if type == "object" and ."$ref" != null then
			{"$ref"}
		else . end)') docs/api_examples.json |
	jq -n 'input * input' > docs/api_service_w_examples.swagger.json
