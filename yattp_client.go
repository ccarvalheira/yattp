package yattp

import (
	"github.com/hashicorp/yamux"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
)

// A YaClient is an object that handles the transport aspects of connecting to a yamux server.
// It seamlessly integrates with Golang's HTTP mechanisms, as it embeds an *http.Client object,
// even tough the underlying transport is made using yamux.
type YaClient struct {
	addr        string
	persistConn *yamux.Session
	connMutex   sync.Mutex
	*http.Client
}

// Creates a new YaClient. Will lazily connect to addr and keep the TCP connection open indefinitely.
func NewYaClient(addr string) (*YaClient, error) {

	y := &YaClient{addr, nil, sync.Mutex{}, nil}

	yamuxDial := func(network, addr string) (net.Conn, error) {
		y.connMutex.Lock()
		if y.persistConn == nil {

			conn, err := net.Dial("tcp", addr)
			if err != nil {
				log.Println("could not dial", err)
				return nil, err
			}
			sess, err := yamux.Client(conn, nil)
			if err != nil {
				log.Println("could not start connection because", err)
				return nil, err
			}
			y.persistConn = sess
		}
		y.connMutex.Unlock()
		return y.persistConn.Open()
	}

	transport := &http.Transport{Dial: yamuxDial}
	y.Client = &http.Client{Transport: transport}

	return y, nil
}

// DoRead is a convenience function for making a request using yamux.
// It builds the request according to the parameters passed in.
func (y *YaClient) DoRead(method, uri string, headers http.Header) (*http.Response, error) {
	var dummyHTTPBody io.Reader
	req, err := http.NewRequest(method, "http://"+y.addr+uri, dummyHTTPBody)
	if err != nil {
		log.Println("could not create request because", err)
		return nil, err
	}
	for h, v := range headers {
		for _, vv := range v {
			req.Header.Add(h, vv)
		}
	}
	resp, err := y.Do(req)
	if err != nil {
		log.Println("something happened with yattp request", err)
		return nil, err
	}
	return resp, nil
}

//This is just a utility function for reading the response body
// because I seem to always need to check the docs on how to do it.
func ReadResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
