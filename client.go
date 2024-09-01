package main

import (
	"log"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/lesismal/nbio"
)

var (
	idleConns = &atomic.Int32{}
)

func StartClientTunnel(taddr, daddr string) (*nbio.Engine, *nbio.Engine) {
	idleConns.Store(0)

	tunnelEngine := nbio.NewEngine(nbio.Config{
		Network:            "tcp",
		MaxWriteBufferSize: 6 * 1024 * 1024,
		NPoller:            runtime.NumCPU(),
	})

	destEngine := nbio.NewEngine(nbio.Config{
		Network:            "tcp",
		MaxWriteBufferSize: 6 * 1024 * 1024,
		NPoller:            runtime.NumCPU(),
	})

	tunnelEngine.OnData(func(tunnelConn *nbio.Conn, data []byte) {
		var destConn *nbio.Conn

		destConn, _ = tunnelConn.Session().(*nbio.Conn)
		if destConn == nil {
			destConn, err := nbio.DialTimeout("tcp", daddr, TIMEOUT)
			if err != nil {
				log.Printf("error in connect to dest: %v\n", err)
				tunnelConn.Close()
				return
			}

			destConn.OnData(func(dConn *nbio.Conn, data []byte) {
				_, err := tunnelConn.Write(data)
				if err != nil {
					tunnelConn.Close()
					dConn.Close()
					return
				}

				tunnelConn.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
				dConn.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
			})

			destEngine.AddConn(destConn)

			tunnelConn.SetSession(destConn)

			idleConns.Add(-1)
		}

		_, err := destConn.Write(data)
		if err != nil {
			tunnelConn.Close()
			destConn.Close()
			return
		}

		tunnelConn.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
		destConn.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
	})

	destEngine.OnData(func(c *nbio.Conn, data []byte) {
		c.DataHandler()(c, data)
	})

	tunnelEngine.OnClose(func(c *nbio.Conn, _ error) {
		sess, ok := c.Session().(*nbio.Conn)
		if ok {
			sess.Close()
		}
	})

	err := tunnelEngine.Start()
	if err != nil {
		log.Fatalln(err)
	}
	err = destEngine.Start()
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		for {
			if idleConns.Load() >= MAX_IDLE_CONNS {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			go func() {
				conn, err := nbio.DialTimeout("tcp", taddr, TIMEOUT)
				if err != nil {
					log.Printf("connect to iran error: %v\n", err)
					return
				}
				tunnelEngine.AddConn(conn)

				idleConns.Add(1)
				log.Printf("idle conns: %v\n", idleConns.Load())
			}()

			time.Sleep(5 * time.Millisecond)
		}
	}()

	return tunnelEngine, destEngine
}
