d=github.com/qiwitech/qdp

mockgen -imports .=github.com/qiwitech/qdp/pt -package mocks -source pt/pt.go > mocks/pt_mock.go
#mockgen -package mocks -source pusher/remotepusher/client.go  > mocks/remotepusher_client_mock.go
mockgen -package mocks $d/proto/gatepb ProcessorServiceInterface > mocks/gate_service_mock.go
mockgen -package mocks $d/proto/plutodbpb PlutoDBServiceInterface  > mocks/plutodb_mock.go
mockgen -package mocks $d/proto/metadbpb MetaDBServiceInterface  > mocks/metadb_mock.go
mockgen -package mocks $d/proto/pusherpb PusherServiceInterface  > mocks/pusher_mock.go
