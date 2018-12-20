# RPC

There are two rpc formats are used.

## API RPC
Simple HTTP+JSON is used from the outer side of API Gates. Each response has `Status` field with status code and error message.

API Gate references could be found here: <link>

## Inner RPC
Custom RPC framework is used for communications between services. It's raw protobuf messages over TCP/IP socket.
It's designed to handle lots of parallel requests with a little of memory usage and allocations. Each request and response is a message, they are not queued, so each request could be sent, response read and processed independently of others.