# Cluster and Routing

System consists of three layers of services: PlutoAPI, Plutos and PlutoDB (or sqldb). Each layer uses one below.

## DB

The Bottom layer is an reliable and durable database that is fault tolerant itself. It just stores transactions and returns them as response on few simple requests.

The next layer uses single balanced address to access some database, like amazon or google cloud balancers.

## Plutos

The Middle layer is processing of transactions.
It's responsible for validating requests, generating new transactions and pushing them to the plutodb.
Plutos nodes are lightweight and don't keep any state when shutted down.

### Sharding

All nodes divide whole set of accounts. Only one node is responsible for each account at any time. Each node processes transactions for every single account sequentially. So double spending is impossible.
There are several ways to guarantee that only one node is responsible for each account at any time.
Simples one is when account range is divided in advance and all routes are not changed while system is working.
This approach has disadvantage: if some plutos fails nobody will process requests from their range of responsible accounts. But if you have many of plutoses than each will own a small part of account and it could be restarted relatively fast.
More complex approach is to have some failover system, that can rearrange accounts between available nodes with guaranties defined above.

### Routing

When node receives some request it checks if this account is owned by it. If it doesn't node returns error with actual routing table. This allows not having any external service for routing.
Also it allows to update routing table at all nodes without isolation requirement and even atomicity could be violated for some period of time without significant hazard. All you risk is some small number of out of service responses, but not integrity or durability.

### Consistency

Since accounts are spread across the cluster it's possible that sender and receiver are owned by different nodes.
This is not the problem because transfer is not some complex transaction at multiple nodes when value is subtracted from one account and added to another.
Transfer is an single piece of data.
All the calculations are made at single (sender's) node and pushed into db that is (reliable, we remember) the single point of truth.
If some failure happens we erase failed node (or just single account) and fetch account chain tail from db.
If transaction was written, transaction has succeed, if not client must retry and it will be succeed then.
Retrying makes the process smooth for client, did some failure happen or not.
Any way request is committed into db or not, money was sent or not, no intermediate state is possible.
All requests are idempotent so there is no hazard in retrying them.
And you can't spend your money twice since each request contains prev_hash that is unique for every transfer.

### Push

Since sender and receiver account could be at different nodes we have to deliver transaction to receiver's node somehow.
Easiest way is to push needed transactions directly to receivers node just after they were committed into db.
If push to other node is failed client must retry, and each request is idempotent, remember?
Even if we committed transactions into db but not pushed them to receiver's node it gets them from db, but not immediately.

All that process is hidden under `Pusher` interface.
So plutos works equally if we use database of not, push transactions to only receivers nodes or to some third party service also.

## API

Top layer is an external API. This service is called [PlutoAPI](PlutoAPI.md). It doesn't keep any state when shutted down. It doesn't communicate with other PlutoAPIs.

The only tasks of this service are to get user request, convert it into inner RPC format and pass request to the right plutos node.

When started plutoapi chooses random plutos to route request to and receives actual routing table as described at [Routing section](#Routing).
The same happens if cluster was changed
