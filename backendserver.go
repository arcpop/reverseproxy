package reverseproxy

import (
    "net/http"
	"net"
	"time"
	"bufio"
	"fmt"
	"io"
)

type BackendServer interface {
    //Forwards a request to the backend server.
    SendRequest(r *http.Request) (*http.Response, error)
    //Returns the percentual load on the server, from 0.0 - 1.0
    GetLoad() float32
    
    GetHostPort() string
}

type backendServer struct {
    address *net.TCPAddr
    connectionPool chan net.Conn
    connectionFactory func () (net.Conn, error)
    usedSlots, availableSlots uint32
    hostPort string
}

func NewBackendServer(hostPort string, numberOfConnections int, keepAlive bool) (BackendServer, error) {
    addr, err := net.ResolveTCPAddr("tcp4", hostPort)
    if err != nil {
        return nil, err
    }
    bs := &backendServer{
        hostPort: hostPort,
        address: addr,
        connectionPool: make(chan net.Conn, numberOfConnections),
        connectionFactory: func () (net.Conn, error)  {
            c, err := net.DialTCP("tcp4", nil, addr)
            if err != nil {
                return nil, err
            }
            if keepAlive {
                err = c.SetKeepAlive(keepAlive)
                if err != nil {
                    c.Close()
                    return nil, err
                }
                c.SetKeepAlivePeriod(30 * time.Second)
                if err != nil {
                    c.Close()
                    return nil, err
                }
            }
            return c, nil
        },
        availableSlots: uint32(numberOfConnections),
    }
    
    for i := 0; i < numberOfConnections; i++ {
        var c net.Conn
        c, err = bs.connectionFactory()
        if err != nil {
            //Perform no cleanup, something is probably broken...
            return nil, err
        }
        bs.connectionPool <- c
    }
    
    return bs, nil
}

func (b *backendServer) GetHostPort() string {
    return b.hostPort
}

//Forwards a request to the backend server.
func (b *backendServer) SendRequest(r *http.Request) (*http.Response, error) {
    var err error
    conn := (<- b.connectionPool).(*net.TCPConn)
    fmt.Printf("%v\n", r.Header)
    err = r.Header.Write(conn)
    if err != nil {
        fmt.Printf("Error Header %v\n", err)
        return nil, err
    }
    if r.ContentLength > 0 {
        body := make([]byte, r.ContentLength)   
        _, err := io.ReadFull(r.Body, body)
        if err != nil {
            fmt.Println("Error 0: ", err)
            return nil, err
        }
        err = r.Body.Close()
        if err != nil {
            fmt.Println("Error 1: ", err)
            return nil, err
        }
        
        for i := 0; i < len(body); {
            n, err := conn.Write(body[i:])
            if err != nil {
                conn.Close()
                conn2, err2 := b.connectionFactory()
                for ; err2 != nil; conn2, err2 = b.connectionFactory() {}
                b.connectionPool <- conn2
                fmt.Println("Error 2: ", err)
                return nil, err
            }
            i += n
        }
    }
    resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
    if err != nil {
        fmt.Printf("%T %+v\n", err, err)
        conn.Close()
        conn2, err2 := b.connectionFactory()
        for ; err2 != nil; conn2, err2 = b.connectionFactory() {}
        b.connectionPool <- conn2
        return nil, err
    }
    b.connectionPool <- conn
    return resp, nil
}

//Returns the percentual load on the server, from 0.0 - 1.0
func (b *backendServer) GetLoad() float32 {
    used := float32(b.usedSlots)
    available := float32(b.availableSlots)
    if available >= used {
        return 1.0
    } else if available <= 0.0 {
        return 0.0
    }
    return used / available
}
