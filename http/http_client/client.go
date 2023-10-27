package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	sal "github.com/salrashid123/sts/http"
	"golang.org/x/oauth2"
)

var (
	stsaddress  = flag.String("stsaddress", "https://stsserver-3kdezruzua-uc.a.run.app/token", "STS Server address")
	stsaudience = flag.String("stsaudience", "stsserver-3kdezruzua-uc.a.run.app", "the audience and resource value to send to STS server")
	scope       = flag.String("scope", "https://www.googleapis.com/auth/cloud-platform", "scope to send to STS server")
	endpoint    = flag.String("endpoint", "http://localhost:8080/", "the server to send the exchanged bearer token to (either http://localhost:8080/ or https://httpbin.org/get")
)

const (
	secret = "iamtheeggman"
)

func main() {
	flag.Parse()

	rootTS := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: secret,
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Duration(time.Second * 60)),
	})

	stsTokenSource, _ := sal.STSTokenSource(
		&sal.STSTokenConfig{
			TokenExchangeServiceURI: *stsaddress,
			Resource:                *stsaudience,
			Audience:                *stsaudience,
			Scope:                   *scope,
			SubjectTokenSource:      rootTS,
			SubjectTokenType:        "urn:ietf:params:oauth:token-type:access_token",
			RequestedTokenType:      "urn:ietf:params:oauth:token-type:access_token",
			HTTPClient:              http.DefaultClient,
		},
	)

	tok, err := stsTokenSource.Token()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("New Token: %s", tok.AccessToken)

	client := oauth2.NewClient(context.TODO(), stsTokenSource)
	resp, err := client.Get(*endpoint)
	if err != nil {
		log.Printf("Error creating client %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error connecting to server %v", http.StatusText(resp.StatusCode))
		return
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	bodyString := string(bodyBytes)
	log.Printf("%s", bodyString)

}
