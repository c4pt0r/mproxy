package proxy

import (
	"log"
	"net"
	"runtime"
)

type Server struct {
	running  bool
	listener net.Listener
	addr     string
	dbConn   *DbConn
}

func NewServer(cfg *config) *Server {
	s := new(Server)
	s.addr = cfg.addr
	s.dbConn = NewDbConn(cfg)
	return s
}

func (s *Server) Start() error {
	var err error
	log.Println(s.addr)
	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		log.Println("listen socket error...%s", err.Error())
		return err
	}
	s.running = true
	for s.running {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Println("accept error")
			continue
		}
		go s.onConn(conn)
	}
	return nil
}

func (s *Server) onConn(c net.Conn) {
	log.Println("on client")
	conn := newClient(s, c)
	defer func() {
		if err := recover(); err != nil {
			const size = 4096
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Println("onConn panic %v: %v\n%s", c.RemoteAddr().String(), err, buf)
		}
		conn.Close()
	}()
	if err := conn.Handshake(); err != nil {
		// TODO
	}
	conn.Serve()
}

func (s *Server) Stop() {
	s.running = false
	if s.listener != nil {
		s.listener.Close()
	}
}
