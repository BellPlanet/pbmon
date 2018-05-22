package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/BellPlanet/pbmon/tpl"
	"github.com/djherbis/stream"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var (
	httpAddr string
)

var (
	server *Server
)

func main() {
	flag.StringVar(&httpAddr, "httpAddr", ":12223", "http listen address")

	flag.Parse()

	setupLogger()
	startServer(httpAddr)
}

func setupLogger() {
	log.SetFlags(log.LstdFlags)
}

func startServer(httpAddr string) {
	server := NewServer(httpAddr)

	server.Start()
}

type Server struct {
	httpMux    *mux.Router
	httpServer *http.Server
	upgrader   *websocket.Upgrader
	stream     *stream.Stream
}

func NewServer(httpAddr string) *Server {
	server := &Server{
		httpMux:  mux.NewRouter(),
		upgrader: &websocket.Upgrader{},
		stream:   stream.NewMemStream(),
	}

	server.httpServer = &http.Server{
		Addr:    httpAddr,
		Handler: server.httpMux,
	}

	server.httpMux.HandleFunc("/stream", server.HandleStream).Methods("GET")
	server.httpMux.HandleFunc("/", server.HandleIndex).Methods("GET")

	return server
}

func (s Server) Start() error {
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
		_, err := reader.Read(buf)
		if err != nil {
			log.Printf("read stream failed: %+v", err)
			return
		}

		// TODO: read meta
		if err := c.WriteMessage(websocket.BinaryMessage, buf); err != nil {
			log.Printf("write message failed: %+v", err)
			return
		}
	}
}

func (s Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	tpl.Index.Execute(w, nil)
}
