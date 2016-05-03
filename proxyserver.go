package reverseproxy


import (
    "net/http"
	"io/ioutil"
	"log"
)

type BalancingMode func () ServerPool

var (
    BalancingRoundRobin BalancingMode = NewRoundRobin
    BalancingLowestLoad BalancingMode = NewLowestLoad
)

type ReverseProxy struct {
    Pool ServerPool
    NumberOfServers int
}

func NewReverseProxy(servers []string, numberOfConnections int, backendKeepAlive bool, balancingMode BalancingMode) (*ReverseProxy, error) {
    numberOfServers := len(servers)
    pool := balancingMode()
    for _, server := range servers {
        bs, err := NewBackendServer(server, numberOfConnections, backendKeepAlive)
        if err != nil {
            return nil, err
        }
        pool.AddServer(bs)
    }
    return &ReverseProxy{
        Pool: pool,
        NumberOfServers: numberOfServers,
    }, nil
}


func (rp *ReverseProxy) ListenAndServe(addr string) error {
    return http.ListenAndServe(addr, rp)
}

func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request)  {
    server := rp.Pool.GetOptimalServer()
    if server == nil {
        log.Println("Case 1")
        http.Error(w, "500 Internal server error", http.StatusInternalServerError)
        return
    }
    r.Header.Set("Connection", "keep-alive")
    r.Host = server.GetHostPort()
    resp, err := server.SendRequest(r)
    if err != nil {
        http.Error(w, "500 Internal server error", http.StatusInternalServerError)
        return
    }
    body, err := ioutil.ReadAll(resp.Body)
    resp.Body.Close()
    if err != nil {
        log.Println("Case 3: ", err)
        http.Error(w, "500 Internal server error", http.StatusInternalServerError)
        return
    }
    for k, vs := range resp.Header {
        for i, v := range vs {
            if i == 0 {
                w.Header().Set(k, v)
            } else {
                w.Header().Add(k, v)
            }
        }
    }
    w.WriteHeader(resp.StatusCode)
    for n := 0; n < len(body);  {
        i, err := w.Write(body[n:])
        if err != nil {
            log.Println("Case 4: ", err)
            return
        }
        n += i
    }
    log.Println("Case 5")
}