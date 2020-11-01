FROM golang:1.14 as build

ENV GO111MODULE=on
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build sts_server.go

FROM gcr.io/distroless/base
COPY --from=build /app/sts_server /
EXPOSE 8080
ENTRYPOINT ["/sts_server"]