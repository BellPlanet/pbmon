package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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
	flag.StringVar(&pbFile, "pbFile", "pb.proto", "protobuf file")
	flag.StringVar(&pbMessageType, "pbMessageType", "Envelope", "protobuf root messge type")

	flag.Parse()

	if port := os.Getenv("PORT"); port != "" {
		httpAddr = fmt.Sprintf(":%s", port)
	}

	if udp := os.Getenv("UDPADDRESS"); udp != "" {
		udpAddr = udp
	}

	if file := os.Getenv("PBFILE"); file != "" {
		pbFile = file
	}

	if mt := os.Getenv("PBMESSAGE"); mt != "" {
		pbMessageType = mt
	}

	setupLogger()
	pb := mustReadPbFile()
	startServer(httpAddr, udpAddr, pb, pbMessageType)
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

func startServer(httpAddr, udpAddr, pb, pbMessageType string) {
	server, err := NewServer(httpAddr, udpAddr, pb, pbMessageType)
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

	pb            string
	pbMessageType string
}

func NewServer(httpAddr, udpAddr, pb, pbMessageType string) (*Server, error) {
	uAddr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	server := &Server{
		stream: stream.NewMemStream(),

		httpMux:  mux.NewRouter(),
		upgrader: &websocket.Upgrader{},

		udpAddr: uAddr,

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
