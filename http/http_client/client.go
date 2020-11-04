package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	testts "github.com/salrashid123/oauth2/dummy"
	sal "github.com/salrashid123/oauth2/sts"
	"golang.org/x/oauth2"
)

var (
	stsaddress  = flag.String("stsaddress", "https://stsserver-6w42z6vi3q-uc.a.run.app/token", "STS Server address")
	stsaudience = flag.String("stsaudience", "stsserver-6w42z6vi3q-uc.a.run.app", "the audience and resource value to send to STS server")
	scope       = flag.String("scope", "https://www.googleapis.com/auth/cloud-platform", "scope to send to STS server")
)

const (
	secret = "iamtheeggman"
)

func main() {
	flag.Parse()

	client := &http.Client{}

	myts, err := testts.NewDummyTokenSource(&testts.DummyTokenConfig{
		TokenValues:             []string{"iamtheeggman", "iamtheeggman", "someotherpassword"},
		RotationIntervalSeconds: 10,
	})
	if err != nil {
		log.Fatal(err)
	}
	stsTokenSource, _ := sal.STSTokenSource(
		&sal.STSTokenConfig{
			TokenExchangeServiceURI: *stsaddress,
			Resource:                *stsaudience,
			Audience:                *stsaudience,
			Scope:                   *scope,
			SubjectTokenSource:      myts,
			SubjectTokenType:        "urn:ietf:params:oauth:token-type:access_token",
			RequestedTokenType:      "urn:ietf:params:oauth:token-type:access_token",
		},
	)

	tok, err := stsTokenSource.Token()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("New Token: %s", tok.AccessToken)

	for i := 0; i < 100; i++ {
		client = oauth2.NewClient(context.TODO(), stsTokenSource)
		resp, err := client.Get("http://localhost:8080/")
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
		time.Sleep(time.Duration(5 * time.Second))
	}

}
