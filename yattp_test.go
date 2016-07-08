package yattp

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func testMux() *http.ServeMux {

	mux := http.NewServeMux()
	mux.HandleFunc("/cenas", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/cenas" {
			http.NotFound(w, req)
			return
		}
		fmt.Fprintf(w, "Welcome to the home page!")
	})
	return mux
}

func testHTPPServer() *httptest.Server {
	return httptest.NewServer(testMux())
}

func testClient(url string) (*YaClient, error) {
	return NewYaClient(url)
}

func testServer() *YaServer {
	return NewYaServer("127.0.0.1:0", nil, testMux())
}

func testClientServer() (*YaServer, *YaClient) {
	s := testServer()
	go s.ListenAndServe()
	time.Sleep(1 * time.Millisecond)
	cl, _ := testClient(s.Addr)
	return s, cl
}

func TestYa(t *testing.T) {
	s, cl := testClientServer()
	defer s.Close()
	resp, err := cl.DoRead("GET", "/cenas", nil)
	if err != nil {
		t.Error("cannot be bad response", err)
	}
	byteBody, err := ReadResponseBody(resp)
	if err != nil {
		t.Error("some error reading body", err)
	}
	if !bytes.Equal(byteBody, []byte("Welcome to the home page!")) {
		t.Error("message different than expected", err)
	}
}

func TestNoServer(t *testing.T) {
	cl, err := testClient("127.0.0.1:9104")
	_, err = cl.DoRead("GET", "/cenas", nil)
	if err == nil {
		t.Error("must have a connection error because there is no server running")
	}
}

func TestYaMultiple(t *testing.T) {
	s, cl := testClientServer()
	defer s.Close()
	var wg sync.WaitGroup
	wg.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			defer wg.Done()
			resp, err := cl.DoRead("GET", "/cenas", nil)
			if err != nil {
				t.Error("cannot be bad response", err)
			}
			byteBody, err := ReadResponseBody(resp)
			if err != nil {
				t.Error("some error reading body", err)
			}
			if !bytes.Equal(byteBody, []byte("Welcome to the home page!")) {
				t.Error("message different than expected", err)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkYamuxMultipleRequests(b *testing.B) {
	s, cl := testClientServer()
	defer s.Close()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, _ := cl.DoRead("GET", "/cenas", nil)
			byteBody, err := ReadResponseBody(resp)
			if !bytes.Equal(byteBody, []byte("Welcome to the home page!")) {
				log.Println("message different than expected", err)
			}
		}
	})
	b.StopTimer()
}

func TestYamux100RequestsManual(t *testing.T) {
	s, cl := testClientServer()
	defer s.Close()
	var wg sync.WaitGroup
	var total time.Duration
	for j := 0; j < 5; j++ {
		start := time.Now()
		wg.Add(100)
		for i := 0; i < 100; i++ {
			go func() {
				defer wg.Done()
				resp, _ := cl.DoRead("GET", "/cenas", nil)
				byteBody, err := ReadResponseBody(resp)
				if !bytes.Equal(byteBody, []byte("Welcome to the home page!")) {
					log.Println("message different than expected", err)
				}
			}()
		}
		wg.Wait()
		total += time.Since(start)
	}
	log.Println("YATTP 5*100 concurrent requests took", total)
}

func BenchmarkRegularHTTPMultipleRequests(b *testing.B) {
	mx := testMux()
	server := httptest.NewServer(mx)
	defer server.Close()

	client := http.Client{}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var dummyHTTPBody io.Reader
			req, _ := http.NewRequest("GET", server.URL+"/cenas", dummyHTTPBody)
			resp, err := client.Do(req)
			if err != nil {
				log.Println(err)
			} else {
				byteBody, _ := ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				if !bytes.Equal(byteBody, []byte("Welcome to the home page!")) {
					log.Println("message different than expected", err)
				}
			}
		}
	})
	b.StopTimer()
}

func TestRegularHTTP100RequestsManual(t *testing.T) {
	server := httptest.NewServer(testMux())
	defer server.Close()

	client := http.Client{}
	var wg sync.WaitGroup
	var total time.Duration
	for j := 0; j < 5; j++ {
		start := time.Now()
		wg.Add(100)
		for i := 0; i < 100; i++ {
			go func() {
				defer wg.Done()
				var dummyHTTPBody io.Reader
				req, _ := http.NewRequest("GET", server.URL+"/cenas", dummyHTTPBody)
				resp, err := client.Do(req)
				if err != nil {
					log.Println(err)
				} else {
					byteBody, _ := ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					if !bytes.Equal(byteBody, []byte("Welcome to the home page!")) {
						log.Println("message different than expected", err)
					}
				}
			}()
		}
		wg.Wait()
		total += time.Since(start)
	}
	log.Println("Regular HTTP 5*100 concurrent requests took", total)
}
