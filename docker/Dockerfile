FROM golang:1.10.2-alpine

RUN mkdir -p /go/src/github.com/BellPlanet/pbmon
WORKDIR /go/src/github.com/BellPlanet/pbmon
COPY . .
RUN go build -o bin/pbmon ./cmd/pbmon/...

EXPOSE 12223/tcp
EXPOSE 12224/udp

CMD ["/go/src/github.com/BellPlanet/pbmon/bin/pbmon"]
