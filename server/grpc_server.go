package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/salrashid123/sts_server/echo"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

var (
	grpcport = flag.String("grpcport", ":8080", "grpcport")
)

const (
	allowedToken = "iamthewalrus"
)

type server struct{}

func (s *server) SayHello(ctx context.Context, in *echo.EchoRequest) (*echo.EchoReply, error) {

	log.Println("Got rpc: --> ", in.Name)
	md, _ := metadata.FromIncomingContext(ctx)
	if len(md["authorization"]) > 0 {
		reqToken := md["authorization"][0]
		splitToken := strings.Split(reqToken, "Bearer")
		reqToken = strings.TrimSpace(splitToken[1])
		if reqToken != allowedToken {
			return nil, grpc.Errorf(codes.Unauthenticated, "Authorization header value invalid")
		}
	} else {
		return nil, grpc.Errorf(codes.Unauthenticated, "Authorization header not provided")
	}
	h := os.Getenv("K_REVISION")
	return &echo.EchoReply{Message: "Hello " + in.Name + "  from K_REVISION " + h}, nil
}

func main() {

	flag.Parse()

	lis, err := net.Listen("tcp", *grpcport)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	sopts := []grpc.ServerOption{grpc.MaxConcurrentStreams(10)}

	s := grpc.NewServer(sopts...)

	echo.RegisterEchoServerServer(s, &server{})

	log.Println("Starting gRPC server on port :8080")

	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	go func() {
		sig := <-gracefulStop
		log.Printf("caught sig: %+v", sig)
		log.Println("Wait for 1 second to finish processing")
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
	s.Serve(lis)
}
