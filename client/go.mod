module main

go 1.15

require (
	echo v0.0.0
	golang.org/x/net v0.0.0-20201031054903-ff519b6c9102 // indirect
	google.golang.org/grpc v1.33.1 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)

replace (
 	echo => ./echo
)