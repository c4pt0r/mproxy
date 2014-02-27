package proxy

import (
    "log"
    "runtime"
    "net"
    "bufio"
)

type client struct {
    user string
    db   string
    closed bool
    lastInserId int64
    affectedRows int64

    server *Server
    conn net.Conn
}

func newClient(s *Server, conn net.Conn) *client {
    c := new(client)
    c.server = s
    c.conn = conn
    return c
}


func (c *client) Close() error {
    c.conn.Close()
    c.closed = true
    return nil
}

func (c *client) Handshake() error {
    return nil
}

func (c *client) Serve() {
    defer func() {
		r := recover()
		if err, ok := r.(error); ok {
			const size = 4096
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Println("%v, %s", err, buf)
		}
        c.Close()
    }()

    for {
        sr := bufio.NewReader(c.conn)
        if sr != nil {
            line , _, err := sr.ReadLine()
            log.Print(line)
            if err != nil {
                break
            }
        }
    }
}

