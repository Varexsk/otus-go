package internalgrpc

//go:generate protoc --proto_path=../../../api --go_out=../../../ --go_opt=module=github.com/varexsk/hw12_13_14_15_calendar --go-grpc_out=../../../ --go-grpc_opt=module=github.com/varexsk/hw12_13_14_15_calendar ../../../api/calendar/v1/event_service.proto
