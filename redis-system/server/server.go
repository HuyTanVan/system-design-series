package server

import (
	"fmt"
	"redis-clone/command"
	"redis-clone/persistence"
	"redis-clone/store"
	"net"
)

type Server struct {
	listenAddr string
	l          net.Listener
	d          *command.Dispatcher
}

func NewServer(listenAddr string) *Server {
	s := store.NewStore()

	aof, err := persistence.NewAof("persistence/appendonly.aof")

	d := command.NewDispatcher(s, aof)

	if err != nil {
		panic(err)
	}
	return &Server{
		listenAddr: listenAddr,
		d:          d,
	}
}

func (s *Server) Start() error {
	l, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	s.l = l
	// fmt.Println("Listening on", s.listenAddr)
	// conn, err := s.l.Accept() // one connection only
	// if err != nil {
	// 	return err
	// }
	fmt.Println("Listening on", s.listenAddr)
	// handleConn(conn) // no goroutine needed either
	// return nil
	return s.acceptLoop()
}

func (s *Server) acceptLoop() error {
	// accept mutiple connections in a loop
	// go routine to handle each connection concurrently
	for {
		conn, err := s.l.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}
		go handleConn(conn, s.d)
	}
}
