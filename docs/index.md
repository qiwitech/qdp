# Plutos - Distributed financial processing

Plutos is a system that provides reliable storage and fast processing of financial transactions between accounts.

Virtually the system is an set of accounts that can send transfers to each other. Each account has a chain of transactions. Each transaction has these fields: sender, receiver, amount, current account balance, hash of previous transaction of the same account and some more. Also each account has chain of settings. Setting transactions keeps user public key, spending limits.

Technically the system consists of database (sqldb), light processing nodes (plutos) and api gates (plutoapi).

## Beginning
* [Install plutos](Install.md)
* [Quick Start](Quick-Start.md)

## API
* [API Reference](api-reference.html)

## Get to know more about internals
* [Cluster and routing](Cluster.md)
* [Processing Node](Processing-Node.md)
* [PlutoAPI](PlutoAPI.md)
* [RPC](RPC.md)

## Tools
* [Command Line Client described at Quick Start](Quick-Start.md)
* [Bench](Bench.md)

## Testing
* [Jepsen](Jepsen.md)
