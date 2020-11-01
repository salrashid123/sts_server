module main

go 1.15

require (
	golang.org/x/net v0.0.0-20201031054903-ff519b6c9102 // indirect
	google.golang.org/api v0.34.0 // indirect
	google.golang.org/grpc v1.33.1 // indirect
	"echo" v0.0.0
)

replace "echo" => "./echo"