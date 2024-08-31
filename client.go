package main

import (
	"log"
	"runtime"
	"sync"
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
		var sess *nbio.Conn
		wg := sync.WaitGroup{}

		sess, _ = tunnelConn.Session().(*nbio.Conn)
		if sess == nil {
			wg.Add(1)
			err := destEngine.DialAsyncTimeout("tcp", daddr, TIMEOUT, func(dstConn *nbio.Conn, err error) {
				if err != nil {
					log.Printf("error in connect to dest: %v\n", err)
					tunnelConn.Close()
					wg.Done()
					return
				}

				dstConn.OnData(func(dConn *nbio.Conn, data []byte) {
					_, err := tunnelConn.Write(data)
					if err != nil {
						tunnelConn.Close()
						dConn.Close()
						return
					}

					tunnelConn.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
					dConn.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
				})

				tunnelConn.SetSession(dstConn)
				sess = dstConn

				idleConns.Add(-1)
				wg.Done()
			})
			wg.Wait()
			if err != nil {
				log.Printf("error in connect to dest: %v\n", err)
				tunnelConn.Close()
				return
			}
		}

		_, err := sess.Write(data)
		if err != nil {
			tunnelConn.Close()
			sess.Close()
			return
		}

		tunnelConn.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
		sess.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
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

	go func(tunnelEngine *nbio.Engine) {
		for {
			if idleConns.Load() >= MAX_IDLE_CONNS {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			tunnelEngine.DialAsyncTimeout("tcp", taddr, TIMEOUT, func(c *nbio.Conn, err error) {
				if err != nil {
					log.Printf("connect to iran error: %v\n", err)
					return
				}

				idleConns.Add(1)
				log.Printf("idle conns: %v\n", idleConns.Load())
			})

			time.Sleep(1 * time.Millisecond)
		}
	}(tunnelEngine)

	return tunnelEngine, destEngine
}
