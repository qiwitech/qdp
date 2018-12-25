# Install

There are number of ways to have processing in your environment. Let's discuss them here.
All the ways doesn't require any dependencies, all binaries are static compiled. So installation process is quiet easy.

## docker

The most simple way if you have a docker is to run docker-compose out of plutos repo. It will start cluster with 3 plutos nodes, single plutoapi and single sqldb with mysql.
```
docker-compose up
```

## Debian package
Second way is to download and install debian package `plutos-1.0.deb`. It contains all the needed binaries.

Another package `plutos-systemd-base-1.0.deb` contains systemd base config files.
This config will start the one node of plutos processor and one of plutoapi. One can set up PLUTOS_NODES and PLUTOS_SELF variables in the `/usr/lib/plutos/plutos.env` environment file to connect multiple nodes together.

Example environment file
```
PLUTOS_SELF=%H:31337
PLUTOS_NODES=hosta:313337,hostb:31337,hostc:31337
```

## Building from sources

If you have go environment installed you can build all binaries from sources by command
```
go install ./cmd/...
```

If you are developer and you want to test some changes, the most convenient way would be to use `goreman` tool to start predefined configuration
```
goreman start
```
It will start 3 plutos nodes with single plutoapi
