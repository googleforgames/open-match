package main

import (
	"fmt"
	"log"
	"net"
	"strings"
)

type server struct {
	pc net.PacketConn

	subscribers map[string]*net.Addr

	dying bool
	exitC chan struct{}
}

func newServer(pc net.PacketConn) *server {
	return &server{
		pc:          pc,
		subscribers: make(map[string]*net.Addr),
		exitC:       make(chan struct{}),
	}
}

func (s *server) exit() {
	close(s.exitC)
}

func (s *server) subscribe(id string, addr *net.Addr) {
	s.subscribers[id] = addr
}

func (s *server) unsubscribe(id string) {
	delete(s.subscribers, id)
}

func (s *server) read(buf []byte) (net.Addr, string, error) {
	n, addr, err := s.pc.ReadFrom(buf)
	if err != nil {
		return addr, "", fmt.Errorf("Error reading: %s", err)
	}

	msg := strings.TrimSpace(string(buf[:n]))
	return addr, msg, nil
}

func (s *server) send(id string, msg string) error {
	log.Printf("Sending message \"%v\" to %s", msg, id)
	b := []byte(msg)

	addr := s.subscribers[id]
	_, err := s.pc.WriteTo(b, *addr)
	if err != nil {
		return fmt.Errorf("Error sending message: %s", err)
	}
	return nil
}

func (s *server) broadcast(msg string) error {
	log.Printf("Broadcasting message \"%v\"", msg)
	b := []byte(msg)
	for _, addr := range s.subscribers {
		_, err := s.pc.WriteTo(b, *addr)
		if err != nil {
			return fmt.Errorf("Error broadcasting message: %s", err)
		}
	}
	return nil
}
