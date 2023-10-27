# go1.19.8 linux/amd64
FROM docker.io/golang@sha256:9f2dd04486e84eec72d945b077d568976981d9afed8b4e2aeb08f7ab739292b3 as build

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build sts_server.go

FROM gcr.io/distroless/base
COPY --from=build /app/sts_server /
EXPOSE 8080
ENTRYPOINT ["/sts_server"]