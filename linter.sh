#!/usr/bin/env bash

for l in $(gometalinter -i | tail -n +2); do
	echo running $l...
	gometalinter --deadline 10s --disable-all -E $l ... &&
		echo no warnings ||
		find -mindepth 1 -not -path './.*' -type d -exec \
			gometalinter --deadline 30s --disable-all -E $l {} \;
done | grep -v "should have comment"
