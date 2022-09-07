module main

go 1.15

require (
	github.com/salrashid123/sts_server/echo v0.0.0
	golang.org/x/net v0.0.0-20201031054903-ff519b6c9102
	google.golang.org/grpc v1.49.0
	github.com/salrashid123/sts_server/sts v0.0.0
)

replace (
	github.com/salrashid123/sts_server/echo => ./echo
	github.com/salrashid123/sts_server/sts => ../sts
)
