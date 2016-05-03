package reverseproxy

import (
	"sync/atomic"
)


type ServerPool interface {
    GetOptimalServer() BackendServer
    AddServer(bs BackendServer)
}

type serverPool struct {
    servers []BackendServer
}

type lowestLoad struct {
    pool serverPool  
    numServers uint64  
}

func NewLowestLoad() ServerPool {
    return &lowestLoad{
        pool: serverPool {
            servers: make([]BackendServer, 0, 10),
        },
    }
}
func (ll *lowestLoad) GetOptimalServer() BackendServer {
    var best float32 = 2.0
    var bestServer BackendServer
    for _, s := range ll.pool.servers {
        l := s.GetLoad()
        if l < best {
            best = l
            bestServer = s
        }
    }
    return bestServer
}
func (ll* lowestLoad) AddServer(b BackendServer)  {
    ll.pool.servers = append(ll.pool.servers, b)
    atomic.AddUint64(&ll.numServers, 1)
}

type roundRobin struct {
    pool serverPool
    counter uint64
    numServers uint64
}

func NewRoundRobin() ServerPool {
    return &roundRobin{
        pool: serverPool {
            servers: make([]BackendServer, 0, 10),
        },
        counter: 0,
        numServers: 0,
    }
}

func (rr *roundRobin) GetOptimalServer() BackendServer {
    val := atomic.AddUint64(&rr.counter, 1)
    return rr.pool.servers[val % rr.numServers]
}
func (rr* roundRobin) AddServer(b BackendServer)  {
    rr.pool.servers = append(rr.pool.servers, b)
    atomic.AddUint64(&rr.numServers, 1)
}