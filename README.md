## Serverless Secure Token Exchange Server(STS) and gRPC STS credentials


This repo contains a very basic STS server deployed on [Cloud Run](https://cloud.google.com/run/docs) which exchanges a one `access_token` for another....basically, a token broker described here:

- [rfc8693: OAuth 2.0 Token Exchange](https://tools.ietf.org/html/rfc8693)

The STS server here is *not* a reference implementation..just use this as a helloworld tutorial

This particular STS server exchanges one static access token for another.   It will exchange 

* `iamtheeggman`  for  `iamthewalrus`   (right, thats it..)


You can use an http client `curl` to see the exchange directly and then use a new gRPC client which utilizes its own gRPC STS Credential object:

- ["google.golang.org/grpc/credentials/sts"](https://godoc.org/google.golang.org/grpc/credentials/sts)

>> This is not an officially supported Google product

---

This tutorial will deploy 

* `gRPC server` on Cloud Run that will inspect the `Authoriztion` header and only allow the request if the value is `iamthewalrus`
* `STS server` on Cloud Run that will only provide the token exchange if the inbound token is `iamtheeggman`
* `gRPC client` that will use the STS Credential object to access the gRPC server after it performs the exchange.


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

gcloud beta run deploy grpcserver  \
  --image gcr.io/$PROJECT_ID/grpc_server \
  --allow-unauthenticated  --region us-central1  --platform=managed  -q
```

#### Deploy STS Server

```bash
cd sts_server/
# use cloud build
# gcloud builds submit --machine-type=n1-highcpu-8 --tag gcr.io/$PROJECT_ID/sts_server . 

# or directly
docker build -t gcr.io/$PROJECT_ID/sts_server .
docker push gcr.io/$PROJECT_ID/sts_server

gcloud beta run deploy stsserver  --image gcr.io/$PROJECT_ID/sts_server \
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

go run grpc_client.go \
  --address $GRPC_SERVER_ADDRESS:443 \
  --cacert googleCA.crt \
  --servername $GRPC_SERVER_ADDRESS \
  --stsaddress $STS_URL \
  --usetls \
  --stsCredFile /tmp/stscreds/creds.txt

## or via client docker image
docker build -t gcr.io/$PROJECT_ID/grpc_client .
docker run -v /tmp/stscreds:/stscreds gcr.io/$PROJECT_ID/grpc_client \
 --address $GRPC_SERVER_ADDRESS:443 \
 --cacert googleCA.crt --servername $GRPC_SERVER_ADDRESS \
 --usetls --stsCredFile /stscreds/creds.txt
```

ok, how does this work with gRPC?  Well you just have to specify the STS Credential type as credential object with the specifications of the STS configurations.

In the command set below, we're specifying the rest endpoint of the STS server, and critically the `SubjectTokenPath` which is the path to the file where we saved the source token.

Once you specify all this, the grpc Client will do the legwork to exchange the source token for the remote one

```golang
import (
  	"google.golang.org/grpc/credentials/sts"
)

		stscreds, err := sts.NewCredentials(sts.Options{
			TokenExchangeServiceURI: *stsaddress,
			Resource:                *stsaudience,
			Audience:                *stsaudience,
			Scope:                   *scope,
			SubjectTokenPath:        *stsCredFile,
			SubjectTokenType:        "urn:ietf:params:oauth:token-type:access_token",
			RequestedTokenType:      "urn:ietf:params:oauth:token-type:access_token",
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
		})
```


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
    "resource": "grpcserver-6w42z6vi3q-uc.a.run.app",
    "audience": "grpcserver-6w42z6vi3q-uc.a.run.app",
    "requested_token_type": "urn:ietf:params:oauth:token-type:access_token",
    "subject_token": "iamtheeggman",
    "subject_token_type": "urn:ietf:params:oauth:token-type:access_token"
}
```

If you change the value of the `subject_token`, the STS server will reject the request.

if you want to, edit the  files and replace the values for `resource` and `audience` with `$GRPC_SERVER_ADDRESS` (its a no-op since this isn't used in this STS server )
