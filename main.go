package main

import (
	"html"
	"fmt"
	"net/http"
	"log"
    "github.com/arcpop/reverseproxy"
	"os"
)
type myHandler struct {
    
}
func runServer(addr string)  {
    s := &http.Server{
        Addr: addr,
        ReadTimeout: 0,
        WriteTimeout: 0,
        Handler: &myHandler{},
    }
    log.Fatal(s.ListenAndServe())
}

func (*myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(os.Stdout, "Header: %v\n", r.Header)
    fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
}

func main()  {
    

    go runServer(":8000")
    go runServer(":8001")
    go runServer(":8002")
    
    rp, err := reverseproxy.NewReverseProxy(
        []string{
            "localhost:8000", 
            "localhost:8001", 
            "localhost:8002",
        }, 
        10,
        true,
        reverseproxy.BalancingRoundRobin)
        
    if err != nil {
        log.Fatal(err)
    }
    log.Fatal(rp.ListenAndServe(":8100"))
}