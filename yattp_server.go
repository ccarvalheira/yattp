package yattp

import (
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/hashicorp/yamux"
)

var (
	// Error returned when the server is gracefully stopped.
	ErrServerStopped = errors.New("server stopped")
)

// A YaServer is a struct that is capable of serving HTTP-like requests being transported by yamux layer.
// It is meant to be used in conjunction with YaClient
type YaServer struct {
	conf     *yamux.Config
	stopChan chan bool
	*http.Server
}

// Creates a new YaServer.
// The config may be nil, in which case a sensible default will be used.
func NewYaServer(addr string, conf *yamux.Config, mux http.Handler) *YaServer {
	if conf == nil {
		//we create some default configuration
		conf = yamux.DefaultConfig()
		conf.AcceptBacklog = 2048
		conf.KeepAliveInterval = 100 * time.Millisecond
		conf.ConnectionWriteTimeout = 250 * time.Millisecond
	}

	srv := YaServer{conf, make(chan bool), &http.Server{Addr: addr, Handler: mux}}
	return &srv
}

// This method overrides the one from http.Server. It handles listening and launching handlers for each connection and request.
func (y *YaServer) ListenAndServe() error {
	ll, err := net.Listen("tcp", y.Addr)
	if err != nil {
		log.Println("wtf tcp listen error", err)
		return err
	}
	y.Addr = ll.Addr().String()
	for {
		ll.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second))
		incoming, err := ll.Accept()
		select {
		case <-y.stopChan:
			return ErrServerStopped
		default:
			//If the channel is still open, continue as normal
		}
		if err != nil {
			netErr, ok := err.(net.Error)
			//If this is a timeout, then continue to wait for
			//new connections
			if ok && netErr.Timeout() && netErr.Temporary() {
				continue
			}
			log.Println("could not accept a connection because", err)
			continue
		}
		yamuxServer, err := yamux.Server(incoming, y.conf)
		if err != nil {
			log.Println("could not start a new mux server because", err)
			continue
		}
		go y.Serve(yamuxServer)
	}
	return nil
}

// Close gracefully stops the server from listening.
// Currently it does not close already opened connections or drains the request backlog.
func (y *YaServer) Close() {
	y.stopChan <- true
}
