## Serverless Security Token Exchange Server(STS) and gRPC STS credentials


This repo contains a very basic STS server deployed on [Cloud Run](https://cloud.google.com/run/docs) which exchanges a one `access_token` for another....basically, a token broker described here:

- [rfc8693: OAuth 2.0 Token Exchange](https://tools.ietf.org/html/rfc8693)

The STS server here is *not* a reference implementation..just use this as a helloworld tutorial

This particular STS server exchanges one static access token for another.   It will exchange 

* `iamtheeggman`  for  `iamthewalrus`   (right, thats it..)


You can use an http client `curl` to see the exchange directly and then use a new gRPC client which utilizes its own gRPC STS Credential object:


>> This is not an officially supported Google product

---

This tutorial will deploy 

* `gRPC server` on Cloud Run that will inspect the `Authoriztion` header and only allow the request if the value is `iamthewalrus`
* `STS server` on Cloud Run that will only provide the token exchange if the inbound token is `iamtheeggman`
* `gRPC client` that will use the STS Credential object to access the gRPC server after it performs the exchange.

then if you want

* `http_server` locally which will inspect the `Authoriztion` header and only allow the request if the value is `iamthewalrus`
* `http_client` uses a the sts client library from `https://github.com/salrashid123/sts/tree/main/http` to perform the exchange and send the final token to an endpoint


first gRPC

### gRPC

Create a file with the original token:

```bash
mkdir /tmp/stscreds
echo -n iamtheeggman > /tmp/stscreds/creds.txt
```


Setup the environment variables and deploy the gRPC and STS servers to cloud run
```bash
export PROJECT_ID=`gcloud config get-value core/project`
export PROJECT_NUMBER=`gcloud projects describe $PROJECT_ID --format='value(projectNumber)'`
```

#### Deploy gRPC Server

```bash
cd server/
docker build -t gcr.io/$PROJECT_ID/grpc_server .
docker push gcr.io/$PROJECT_ID/grpc_server

gcloud  run deploy grpcserver  \
  --image gcr.io/$PROJECT_ID/grpc_server \
  --allow-unauthenticated  --region us-central1  --platform=managed  -q
```

#### Deploy STS Server

```bash
cd sts_server/
docker build -t gcr.io/$PROJECT_ID/sts_server .
docker push gcr.io/$PROJECT_ID/sts_server

gcloud run deploy stsserver  --image gcr.io/$PROJECT_ID/sts_server \
 --region us-central1  --allow-unauthenticated --platform=managed  -q
```

### gRPC Client


First we need to find the assigned addresses for the gRPC server and the STS Server

```bash
cd client/

export GRPC_SERVER_ADDRESS=`gcloud run services describe grpcserver --format="value(status.url)"`
export GRPC_SERVER_ADDRESS=`echo "$GRPC_SERVER_ADDRESS" | awk -F/ '{print $3}'`
echo $GRPC_SERVER_ADDRESS

export STS_URL=`gcloud run services describe stsserver --format="value(status.url)"`/token
echo $STS_URL


export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info

go run grpc_client.go \
  --address $GRPC_SERVER_ADDRESS:443 \
  --cacert googleCA.crt \
  --servername $GRPC_SERVER_ADDRESS \
  --stsaddress $STS_URL \
  --usetls \
  --stsCredFile /tmp/stscreds/creds.txt

## note, you can get googleCA.crt by copying in `openssl s_client $GRPC_SERVER_ADDRESS:443`
```

ok, how does this work with gRPC?  Well you just have to specify the STS Credential type as credential object with the specifications of the STS configurations.

In the command set below, we're specifying the rest endpoint of the STS server, and critically the `SubjectTokenPath` which is the path to the file where we saved the source token.

Once you specify all this, the grpc Client will do the legwork to exchange the source token for the remote one

> Note: i'm using my own sts credential provider `"github.com/salrashid123/sts_server/sts"` which is a fork of `"google.golang.org/grpc/credentials/sts"` until [issue  5611](https://github.com/grpc/grpc-go/pull/5611) is merged 

```golang
import (
	//"google.golang.org/grpc/credentials/sts"
	"github.com/salrashid123/sts_server/sts"
)

		stscreds, err := sts.NewCredentials(sts.Options{
			TokenExchangeServiceURI: *stsaddress,
			Resource:                *stsaudience,
			Audience:                *stsaudience,
			Scope:                   *scope,
			SubjectTokenPath:        *stsCredFile,
			SubjectTokenType:        "urn:ietf:params:oauth:token-type:access_token",
			RequestedTokenType:      "urn:ietf:params:oauth:token-type:access_token",
			HTTPClient:              http.DefaultClient,
		})

		conn, err = grpc.Dial(*address,
			grpc.WithTransportCredentials(ce),
			grpc.WithPerRPCCredentials(stscreds))    
```

.....now...why is specifying the `SubjectTokenPath` a _file_?  I don't know..at the very least it should be `[]byte` or perhaps an `oauth2.TokenSource` which includes the value of the source token.  I'll file a bug about this. In the meantime, the code contained [here](https://gist.github.com/salrashid123/c9c3863a681b5a8f61c0012ae9d01fcc#file-sts-go-L274) is copy of the grpc sts.go file defines and uses `SubjectTokenSource`.

The usage for this in the client would look like 

```golang
		stscreds, err := sts.NewCredentials(sts.Options{
			TokenExchangeServiceURI: *stsaddress,
			Resource:                *stsaudience,
			Audience:                *stsaudience,
			Scope:                   *scope,
			//SubjectTokenPath:        *stsCredFile,
			SubjectTokenSource: oauth2.StaticTokenSource(&oauth2.Token{
				AccessToken: "iamtheeggman",
				Expiry:      time.Now().Add(time.Duration(300 * time.Second)),
			}),			
			SubjectTokenType:        "urn:ietf:params:oauth:token-type:access_token",
			RequestedTokenType:      "urn:ietf:params:oauth:token-type:access_token",
			//HttpClient:     http.DefaultClient,
		})
```

Alternatively, just to test,you can inject a [test TokenSource](https://github.com/salrashid123/oauth2#usage-dummytokensource) instead of the static on.

### curl

If you just want to see the formats accepted by the STS server, you can use the commands shown below


- as `application/x-www-form-urlencoded`

```bash
$ cd curl/
$ curl -s -H "Content-Type: application/x-www-form-urlencoded"  -d @sts_req.txt  $STS_URL | jq '.'
{
  "access_token": "iamthewalrus",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 60
}
```

- as `application/json`

```bash
$ cd curl/
$ curl -s -X POST -H "Content-Type: application/json" -d @sts_req.json  $STS_URL | jq '.'
{
  "access_token": "iamthewalrus",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 60
}
```

Note the two source files contains the specifications of the source token that will get sent to the STS server

```json
{
    "grant_type": "urn:ietf:params:oauth:grant-type:token-exchange",
    "resource": "grpcserver-3kdezruzua-uc.a.run.app",
    "audience": "grpcserver-3kdezruzua-uc.a.run.app",
    "requested_token_type": "urn:ietf:params:oauth:token-type:access_token",
    "subject_token": "iamtheeggman",
    "subject_token_type": "urn:ietf:params:oauth:token-type:access_token"
}
```

If you change the value of the `subject_token`, the STS server will reject the request.

if you want to, edit the  files and replace the values for `resource` and `audience` with `$GRPC_SERVER_ADDRESS` (its a no-op since this isn't used in this STS server )

---

## STSTokenSource Server demo using HTTP Client

The following bootstraps STS Credentials *from* and *to* an [oauth2.TokenSource](https://godoc.org/golang.org/x/oauth2#TokenSource).

This basically means you can inject any token into a TokenSource and then utilize an unsupported library here:
 [https://github.com/salrashid123/oauth2#usage-sts](https://github.com/salrashid123/oauth2#usage-sts)

to derive a new tokensource based off an STS server.

I'm using an oauth2 token source here but clearly, the source and target tokensources need not be oauth2.

>> this is not supported by google and neither is `github.com/salrashid123/oauth2/sts`

You must first deploy the STS Server into cloud run as shown in the root repo.

### Start HTTP Server

```bash
cd http_server
go run server.go
```

### Start Client

```bash
$ go run client.go --stsaddress https://stsserver-3kdezruzua-uc.a.run.app/token --endpoint https://httpbin.org/get

2023/10/27 13:20:14 New Token: iamthewalrus
2023/10/27 13:20:14 {
  "args": {}, 
  "headers": {
    "Accept-Encoding": "gzip", 
    "Authorization": "Bearer iamthewalrus", 
    "Host": "httpbin.org", 
    "User-Agent": "Go-http-client/2.0", 
    "X-Amzn-Trace-Id": "Root=1-653bf14e-6f28f7843667ed806be1e4bc"
  }, 
  "origin": "108.51.25.168", 
  "url": "https://httpbin.org/get"
}

$ go run client.go --stsaddress https://stsserver-3kdezruzua-uc.a.run.app/token --endpoint http://localhost:8080/
2023/10/27 13:20:27 New Token: iamthewalrus
2023/10/27 13:20:27 ok

```

The following snippet shows the exchange happening from a source token "iamtheeggman" to a destination tokensource that will automatically perform the STS Exchange.


One the exchange takes place, a plain authorized request 
```golang
import (
   	sal "github.com/salrashid123/sts/http"
)

	client := &http.Client{}

    // start with a source tken
	rootTS := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "iamtheeggman",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Duration(time.Second * 60)),
    })
    
    // exchange it
	stsTokenSource, _ := sal.STSTokenSource(
		&sal.STSTokenConfig{
			TokenExchangeServiceURI: "https://https://stsserver-3kdezruzua-uc.a.run.app/token",
			Resource:                "localhost",
			Audience:                "localhost",
			Scope:                   "https://www.googleapis.com/auth/cloud-platform",
			SubjectTokenSource:      rootTS,
			SubjectTokenType:        "urn:ietf:params:oauth:token-type:access_token",
			RequestedTokenType:      "urn:ietf:params:oauth:token-type:access_token",
			//HttpClient:     http.DefaultClient,
		},
    )
    
    // print the new token (iamthewalrus)
	tok, err := stsTokenSource.Token()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("New Token: %s", tok.AccessToken)    

    // use the new token
	client = oauth2.NewClient(context.TODO(), stsTokenSource)
	resp, err := client.Get("http://localhost:8080/")
	if err != nil {
		log.Printf("Error creating client %v", err)
		return
    }
```
