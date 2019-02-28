[![godoc](https://godoc.org/github.com/qiwitech/qdp?status.svg)](https://godoc.org/github.com/qiwitech/qdp)
[![Build Status](https://travis-ci.org/qiwitech/qdp.svg?branch=master)](https://travis-ci.org/qiwitech/qdp/builds)
[![Go Report Card](https://goreportcard.com/badge/github.com/qiwitech/qdp)](https://goreportcard.com/report/github.com/qiwitech/qdp)
![GitHub](https://img.shields.io/github/license/qiwitech/qdp.svg)
[![MicroBadger Size](https://img.shields.io/microbadger/image-size/qiwitech/qdp.svg)](https://hub.docker.com/r/qiwitech/qdp)

# QDP (aka Plutos)

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
Althrough it's not the subject to be published, general approach is described in [AsgardDB: Fast and Scalable Financial Database](https://www.researchgate.net/publication/326816360_AsgardDB_Fast_and_Scalable_Financial_Database)

Simple mysql wrapper is provided to start with `./cmd/sqldb/`.

## Authors

* [Yuri Korzhenevsky R&D Center](https://www.rnd.center)
* [QIWI Blockchain Technologies](https://qiwi.tech)
