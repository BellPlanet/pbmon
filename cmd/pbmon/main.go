package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/BellPlanet/pbmon/tpl"
	"github.com/djherbis/stream"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var (
	httpAddr      string
	pbFile        string
	pbMessageType string
)

var (
	server *Server
)

func main() {
	flag.StringVar(&httpAddr, "httpAddr", ":12223", "http listen address")
	flag.StringVar(&pbFile, "pbFile", "pb.proto", "protobuf file")
	flag.StringVar(&pbMessageType, "pbMessageType", "Envelope", "protobuf root messge type")

	flag.Parse()

	if port := os.Getenv("PORT"); port != "" {
		httpAddr = fmt.Sprintf(":%s", port)
	}

	if file := os.Getenv("PBFILE"); file != "" {
		pbFile = file
	}

	if mt := os.Getenv("PBMESSAGE"); mt != "" {
		pbMessageType = mt
	}

	setupLogger()
	pb := mustReadPbFile()
	startServer(httpAddr, pb, pbMessageType)
}

func setupLogger() {
	log.SetFlags(log.LstdFlags)
}

func mustReadPbFile() string {
	c, err := ioutil.ReadFile(pbFile)
	if err != nil {
		panic(err)
	}
	return string(c)
}

func startServer(httpAddr, pb, pbMessageType string) {
	server := NewServer(httpAddr, pb, pbMessageType)

	server.Start()
}

type Server struct {
	httpMux    *mux.Router
	httpServer *http.Server
	upgrader   *websocket.Upgrader
	stream     *stream.Stream

	pb            string
	pbMessageType string
}

func NewServer(httpAddr, pb, pbMessageType string) *Server {
	server := &Server{
		httpMux:  mux.NewRouter(),
		upgrader: &websocket.Upgrader{},
		stream:   stream.NewMemStream(),

		pb:            pb,
		pbMessageType: pbMessageType,
	}

	server.httpServer = &http.Server{
		Addr:    httpAddr,
		Handler: server.httpMux,
	}

	server.httpMux.HandleFunc("/stream", server.HandleStream).Methods("GET")
	server.httpMux.HandleFunc("/proto", server.HandleProto).Methods("GET")
	server.httpMux.HandleFunc("/proto/settings", server.HandleProtoSettings).Methods("GET")
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

	c.WriteMessage(websocket.BinaryMessage, []byte("hello"))

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

func (s Server) HandleProto(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/vnd.google.protobuf")
	w.Write([]byte(s.pb))
}

func (s Server) HandleProtoSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"message_type": "%s"}`, s.pbMessageType)))
}

func (s Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	tpl.Index.Execute(w, nil)
}
