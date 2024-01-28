FROM golang:1.21-alpine as builder

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /ctf-reverse-proxy cmd/server/main.go

FROM scratch
COPY --from=builder /ctf-reverse-proxy /ctf-reverse-proxy
COPY --from=builder /app/config-example.yaml /config.yaml

EXPOSE 8000 8080
ENTRYPOINT [ "/ctf-reverse-proxy" ]