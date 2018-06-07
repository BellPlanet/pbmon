package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/BellPlanet/pbmon/tpl"
	"github.com/djherbis/stream"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var (
	httpAddr      string
	udpAddr       string
	pbFile        string
	pbMessageType string
)

var (
	server *Server
)

func main() {
	flag.StringVar(&httpAddr, "httpAddr", ":12223", "http listen address")
	flag.StringVar(&udpAddr, "udpAddr", ":12224", "udp listen address")

	flag.Parse()

	if port := os.Getenv("PORT"); port != "" {
		httpAddr = fmt.Sprintf(":%s", port)
	}

	if udp := os.Getenv("UDPADDRESS"); udp != "" {
		udpAddr = udp
	}

	setupLogger()
	startServer(httpAddr, udpAddr)
}

func setupLogger() {
	log.SetFlags(log.LstdFlags)
}

func startServer(httpAddr, udpAddr string) {
	server, err := NewServer(httpAddr, udpAddr)
	if err != nil {
		panic(err)
	}

	server.Start()
}

type Server struct {
	stream *stream.Stream

	httpMux    *mux.Router
	httpServer *http.Server
	upgrader   *websocket.Upgrader

	udpAddr *net.UDPAddr
}

func NewServer(httpAddr, udpAddr string) (*Server, error) {
	uAddr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	server := &Server{
		stream: stream.NewMemStream(),

		httpMux: mux.NewRouter(),
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},

		udpAddr: uAddr,
	}

	server.httpServer = &http.Server{
		Addr:    httpAddr,
		Handler: server.httpMux,
	}

	server.httpMux.HandleFunc("/stream", server.HandleStream).Methods("GET")
	server.httpMux.HandleFunc("/", server.HandleIndex).Methods("GET")

	return server, nil
}

func (s Server) Start() error {
	udpConn, err := net.ListenUDP(s.udpAddr.Network(), s.udpAddr)
	if err != nil {
		return err
	}
	go func(udpConn *net.UDPConn, stream *stream.Stream) {
		log.Printf("udp server started at %s", s.udpAddr)

		io.Copy(stream, udpConn)
	}(udpConn, s.stream)

	log.Printf("http server started at %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s Server) HandleStream(w http.ResponseWriter, r *http.Request) {
	c, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade failed: %+v", err)
		return
	}
	defer c.Close()

	reader, err := s.stream.NextReader()
	if err != nil {
		log.Printf("stream failed: %+v", err)
		return
	}
	defer reader.Close()

	buf := make([]byte, 32*1024)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			log.Printf("read stream failed: %+v", err)
			return
		}

		// TODO: read meta
		if err := c.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
			log.Printf("write message failed: %+v", err)
			return
		}
	}
}

func (s Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	tpl.Index.Execute(w, nil)
}
