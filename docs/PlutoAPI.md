# PlutoAPI

PlutoAPI is an gate between users and the system. All routing is transparent. User can reach only API Gates and not any other service of the system.

Each request is processed in several steps:
1. Node checks arguments on simple errors.
2. If it's history request it's routed to the database.
3. If it's metadata request it's routed to the metadb.
4. Overwise it's transfer request. If metadata is attached to the request it is written to the metadb first under the certain key.
3. Gate finds node that is responsible for requested account.
4. Gate sends request to the responsible node and waits for the response. Most requests are lasts less that 300ms, so it's not as long wait.
5. If routing error is got, routing table is updated by data received from the node and request retried. If more that 3 routing errors taken in a row, error is returned.
6. If response is successful and if metadata was present they are updated with transfer data.
7. Response is returned to the user.