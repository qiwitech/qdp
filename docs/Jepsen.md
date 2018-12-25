# Jepsen

We tested our system with [Jepsen](https://jepsen.io/) - framework for testing distributed systems in failure scenarios.

Testing code is located in [qiwitech/qdp-jepsen](https://github.com/qiwitech/qdp-jepsen) repo.

## Runnig the test

You'll need `docker-compose` to run testing cluster.

```
# build binaries at plutos repo
./build-for-jepsen.sh
# move created archive to jepsen repo (expected that both repositories are located at the same location)
cp plutos.tar ../jepsen/docker/static/
# go to jepsen repo
cd ../jepsen
# run testing cluster
./docker/up.sh
# wait until containers are built and running
```
Open new terminal and type there
```
# login to control node shell
docker exec -it jepsen-control bash
# run test
cd plutos && lein run test
# in the end you should see following message:
# Everything looks good! ヽ(‘ー`)ノ
```
