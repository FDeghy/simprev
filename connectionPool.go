package main

import (
	"fmt"
	"time"

	"github.com/lesismal/nbio"
)

var (
	tunnelConns = make(chan *nbio.Conn, 256)
)

func GetTunnelConn() *nbio.Conn {
	select {
	case conn := <-tunnelConns:
		return conn
	case <-time.After(5 * time.Second):
		return nil
	}
}

func PutTunnelConn(c *nbio.Conn) error {
	select {
	case tunnelConns <- c:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout")
	}
}
