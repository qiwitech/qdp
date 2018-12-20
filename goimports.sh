#!/usr/bin/env bash

find -type f -name '*.go' -exec \
	goimports -local github.com/qiwitech -w {} \;
