# Processing Node

Processing node is an light service that doesn't persist any information. All nodes divide all range of account into small parts according to capacity of nodes and accounts load.
New node when started connects to cluster and lease some range of accounts and process their transactions.
Nodes could be configured to process ranges statically or to use special failover algorithm which guarantee that no account will be processed by two different nodes at the same time and that each account will be assigned to some node.

## Transaction processing

When node gets new transfer request it process it in several steps:
1. Checks if sender account is handled by this node. If it doesn't error returned with routing table attached.
2. Checks if this account is loaded into local memory. If it doesn't node fetches few last transactions from db: last settings and usually 5 transfer output transactions, each with inputs.
3. Checks balance, sign and all the hashes. If something failed, error is returned.
4. Calculates new balance, transaction hashes, and saves it into local memory.
5. Pushes transactions into database and on nodes where receivers are handled.
7. If push fails account cleaned form local memory and loaded from the database, so database is the main source of the truth.
6. When new transactions are saved oldest in that account chain are removed from local memory, so that RSS memory is kept in the same level dependent only on number of accounts processed and on number of input transactions they have.

When new transactions are got from another node (this node is the receiver node for that transactions), they are saved into local memory, but account is not considered as loaded.

There could be number of transactions in a batch, all they can have different receivers, but they must have the same sender. Batch is processed atomically.