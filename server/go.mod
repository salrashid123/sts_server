module main

go 1.19

require (
	github.com/salrashid123/sts_server/echo v0.0.0
	golang.org/x/net v0.17.0
	google.golang.org/grpc v1.59.0

)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)

replace github.com/salrashid123/sts_server/echo => ./echo
