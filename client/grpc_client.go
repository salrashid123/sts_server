package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"

	pb "github.com/salrashid123/sts_server/echo"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	//"google.golang.org/grpc/credentials/sts"
	sts "github.com/salrashid123/sts/grpc"
)

var (
	address       = flag.String("address", "grpcserver-3kdezruzua-uc.a.run.app:443", "host:port of gRPC server")
	stsaddress    = flag.String("stsaddress", "https://grpcserver-3kdezruzua-uc.a.run.app", "STS Server address")
	stsaudience   = flag.String("stsaudience", "stsserver-3kdezruzua-uc.a.run.app", "the audience and resource value to send to STS server")
	scope         = flag.String("scope", "https://www.googleapis.com/auth/cloud-platform", "scope to send to STS server")
	cacert        = flag.String("cacert", "", "root CA Certificate for TLS")
	sniServerName = flag.String("servername", "grpcserver-3kdezruzua-uc.a.run.app", "SNIServer Name for the server")
	stsCredFile   = flag.String("stsCredFile", "", "File with the original credentials")
	usetls        = flag.Bool("usetls", false, "startup using TLS")
)

type simpleCreds struct {
	Password string
}

func (c *simpleCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"Authorization": "Bearer " + c.Password,
	}, nil
}
func (c *simpleCreds) RequireTransportSecurity() bool {
	return true
}

func main() {

	flag.Parse()

	ctx := context.Background()

	if *stsCredFile == "" {
		log.Fatalf("stsCredFile must be set")
	}

	var conn *grpc.ClientConn
	var err error
	if !*usetls {
		conn, err = grpc.Dial(*address, grpc.WithInsecure())
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
	} else {
		// rootCAs := x509.NewCertPool()
		// var tlsCfg tls.Config
		// pem, err := ioutil.ReadFile(*cacert)
		// if err != nil {
		// 	log.Fatalf("failed to load root CA certificates  error=%v", err)
		// }
		// if !rootCAs.AppendCertsFromPEM(pem) {
		// 	log.Fatalf("no root CA certs parsed from file ")
		// }
		// tlsCfg.RootCAs = rootCAs
		// tlsCfg.ServerName = *sniServerName
		// ce := credentials.NewTLS(&tlsCfg)

		// serverCA, err := ioutil.ReadFile(*cacert)
		// if err != nil {
		// 	log.Fatalf("did not read tlsCA: %v", err)
		// }
		// caCertPool := x509.NewCertPool()
		// caCertPool.AppendCertsFromPEM(serverCA)
		// customClient := &http.Client{
		// 	Transport: &http.Transport{
		// 		TLSClientConfig: &tls.Config{
		// 			ServerName: "stsserver-3kdezruzua-uc.a.run.app",
		// 			RootCAs:    caCertPool,
		// 		},
		// 	}}

		// ### without auth
		// conn, err = grpc.Dial(*address,
		// 	grpc.WithTransportCredentials(ce))
		// if err != nil {
		// 	log.Fatalf("did not connect: %v", err)
		// }

		// ### test direct
		// conn, err = grpc.Dial(*address,
		// 	grpc.WithTransportCredentials(ce),
		// 	grpc.WithPerRPCCredentials(&simpleCreds{
		// 		Password: "iamthewalrus",
		// 	}))
		// if err != nil {
		// 	log.Fatalf("did not connect: %v", err)
		// }

		ce := credentials.NewTLS(&tls.Config{})

		// ### test with sts
		stscreds, err := sts.NewCredentials(sts.Options{
			TokenExchangeServiceURI: *stsaddress,
			Resource:                *stsaudience,
			Audience:                *stsaudience,
			Scope:                   *scope,
			SubjectTokenPath:        *stsCredFile,
			SubjectTokenType:        "urn:ietf:params:oauth:token-type:access_token",
			RequestedTokenType:      "urn:ietf:params:oauth:token-type:access_token",
			HTTPClient:              http.DefaultClient,
			//HTTPClient: customClient,
		})
		if err != nil {
			log.Fatalf("unable to create TokenSource: %v", err)
		}

		conn, err = grpc.Dial(*address,
			grpc.WithTransportCredentials(ce),
			grpc.WithPerRPCCredentials(stscreds))
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}

	}
	defer conn.Close()
	c := pb.NewEchoServerClient(conn)

	r, err := c.SayHello(ctx, &pb.EchoRequest{Name: "unary RPC msg "})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

	log.Printf("RPC Response: %s", r)

}
