[![godoc](https://godoc.org/github.com/qiwitech/qdp?status.svg)](https://godoc.org/github.com/qiwitech/qdp)
[![Build Status](https://travis-ci.org/nikandfor/qdp.svg?branch=master)](https://travis-ci.org/nikandfor/qdp/builds)
[![codecov](https://codecov.io/gh/nikandfor/qdp/branch/master/graph/badge.svg)](https://codecov.io/gh/nikandfor/qdp)
[![codebeat badge](https://codebeat.co/badges/1812545a-2f5c-4f43-98a3-31a26d0bfc38)](https://codebeat.co/projects/github-com-qiwitech-qdp-master)
[![Go Report Card](https://goreportcard.com/badge/github.com/qiwitech/qdp)](https://goreportcard.com/report/github.com/qiwitech/qdp)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/2567/badge)](https://bestpractices.coreinfrastructure.org/projects/2567)


# Plutos (aka QDP)

Plutos is an financial processing 💵 It's designed to replace existing bank systems, make it cheap, reliable and fast ⚡

* Transfers between any accounts with no restrictions. Scalable.
* Transactions are the main part, not balances. Cashflow and relationships are at the first.
* Fast and efficient. No actions without intention.
* Easy to configure and maintain. No caches, no thresholds, no config files (except systemd).
* Easy to use. HTTP API with several endpoints

## Where to start

The best place to start is [documentation](./docs/index.md)

## Repositories

It's the main repository with plutos code itself and commands.

There are also two RPC libraries with protobuf service generators: [tcprpc](https://github.com/qiwitech/tcprpc) and [graceful](https://github.com/qiwitech/graceful)

And there is [Jepsen test](https://github.com/qiwitech/qdp-jepsen).

## Database

Database is important part of the system. It stores transactions durably, so nothing lost if plutos restarts or fails.
Plutos could work without a database, but nothing would be stored persistently and some part of data would be lost if one of plutoses is stopped or crashed.

The special database was developed along with plutos system to acheive maximum performance and reliability.
Althrough it's not the subject to be published, general approach is described in the paper [AsgardDB: Fast and Scalable Financial Database](https://www.researchgate.net/publication/326816360_AsgardDB_Fast_and_Scalable_Financial_Database)

Simple mysql wrapper is provided to start with `./cmd/sqldb/`.

## Contribution

Contributions are welcome. Please follow these guidlines.
1. It's generally better to start with opening issue with description of the bug or feature you want to fix.
1. [Fork](https://help.github.com/articles/fork-a-repo/) a repo, use separate branches for different features/bugs.
1. Always run `gofmt` on your code before committing it. Run linter. It's ok to have some warnings but this will help to find some common mistakes.
1. All new features should have tests.
1. All new and existing tests must pass.
1. Submit a [pull request](https://help.github.com/articles/creating-a-pull-request/)

## Authors

* [Yuri Korzhenevsky Research and Development Center](https://www.rnd.center)
* [QIWI Blockchain Technologies](https://qiwi.tech)
