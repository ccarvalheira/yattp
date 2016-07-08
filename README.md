# yattp
Ever want to make a load of HTTP connections, but multiplexed over a single TCP connection like HTTP2?
How about the client and server behave like an http.Client and http.Server so the API is basically the same?
Enter yattp!

It uses yamux for the transport layer: https://github.com/hashicorp/yamux

## Instalation

Run go get using a suitable GOPATH.
```
go get github.com/hashicorp/yamux
go get github.com/ccarvalheira/yattp
```

## Usage

```
import (
  "net/http"
  "github.com/ccarvalheira/yattp"
)

func main() {
//we make a mux to handle our requests
//you can also use gorillamux or your favourite mux that works with http.Server
  mux := http.NewServeMux()
	mux.HandleFunc("/yattp", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/yattp" {
			http.NotFound(w, req)
			return
		}
		fmt.Fprintf(w, "Hello!")
	})
	// we create the server and pass an address (same format as http.Server), a *yamux.Confing and the mux
	//if you pass nil as the config, some (possibly) sane defaults will be used
  ys := yattp.NewYaServer(":9102", nil, mux)
  go ys.ListenAndServe()
  
  //we now build the client
  // note that we need to pass the whole host:port address (or FQDN)
  yc, err := yattp.NewYaClient("127.0.0.1:9102")
  if err != nil {
    log.Println("something went wrong creating the client")
    return
  }
  resp, err := yc.DoRead("GET", "/yattp", nil) //or pass in some custom headers
  // note that you may also use every method in http.Client
  if err != nil {
    log.Println("something happened while doing a request", err)
  }
  
}

```

## Pros

* You will only ever open a single connection to a host. This helps keep down on opened TCP ports.
* You can use all the stuff from net/http as if this was really HTTP.
* It may be faster (the benchmarks are not really conclusive).

## Cons

* It does not interoperate with "regular" HTTP1 servers/clients.
* It may be slower (the benchmarks are not really conclusive).

## Additional work/known unfinished stuff

* when Closing a server we only stop listening for new connections; already open connections stay open and fully working
* not sure what happens when a server crashes. I don't think the client notices this due to the way the Dial function is implemented. I'd have to look into it.
