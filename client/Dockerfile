# go1.19.8 linux/amd64
FROM docker.io/golang@sha256:9f2dd04486e84eec72d945b077d568976981d9afed8b4e2aeb08f7ab739292b3 as build

RUN apt-get update -y && apt-get install -y build-essential wget unzip curl
RUN curl -OL https://github.com/google/protobuf/releases/download/v3.19.0/protoc-3.19.0-linux-x86_64.zip && \
    unzip protoc-3.19.0-linux-x86_64.zip -d protoc3 && \
    mv protoc3/bin/* /usr/local/bin/ && \
    mv protoc3/include/* /usr/local/include/


WORKDIR /app
ADD . /app
RUN go mod download
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
RUN go install github.com/golang/protobuf/protoc-gen-go@latest
RUN /usr/local/bin/protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=require_unimplemented_servers=false --go-grpc_out=. --go-grpc_opt=paths=source_relative echo/echo.proto
RUN export GOBIN=/app/bin && go install grpc_client.go


FROM gcr.io/distroless/base
COPY --from=build /app/bin/grpc_client /grpc_client
COPY --from=build /app/googleCA.crt /googleCA.crt
EXPOSE 8080
ENTRYPOINT ["/grpc_client"]