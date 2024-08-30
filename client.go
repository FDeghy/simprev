package main

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/lesismal/nbio"
)

var (
	idleConns = &atomic.Int32{}
)

func StartClientTunnel(taddr, daddr string) (*nbio.Engine, *nbio.Engine) {
	idleConns.Store(0)

	tengine := nbio.NewEngine(nbio.Config{
		Network:            "tcp",
		MaxWriteBufferSize: 6 * 1024 * 1024,
	})

	dengine := nbio.NewEngine(nbio.Config{
		Network:            "tcp",
		MaxWriteBufferSize: 6 * 1024 * 1024,
	})

	tengine.OnData(func(c *nbio.Conn, data []byte) {
		var sess *nbio.Conn
		sess, ok := c.Session().(*nbio.Conn)
		if !ok || sess == nil {
			conn, err := nbio.Dial("tcp", daddr)
			if err != nil {
				log.Printf("err: %v\n", err)
				c.Close()
				return
			}

			conn.OnData(func(conn *nbio.Conn, data []byte) {
				_, err := c.Write(data)
				if err != nil {
					c.Close()
					sess.Close()
					return
				}

				c.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
				sess.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
			})
			dengine.AddConn(conn)

			c.SetSession(conn)
			sess = conn

			idleConns.Add(-1)
		}

		_, err := sess.Write(data)
		if err != nil {
			c.Close()
			sess.Close()
			return
		}

		c.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
		sess.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
	})

	dengine.OnData(func(c *nbio.Conn, data []byte) {
		c.DataHandler()(c, data)
	})

	tengine.OnClose(func(c *nbio.Conn, _ error) {
		sess, ok := c.Session().(*nbio.Conn)
		if ok {
			sess.Close()
		}
	})

	err := tengine.Start()
	if err != nil {
		log.Fatalln(err)
	}
	err = dengine.Start()
	if err != nil {
		log.Fatalln(err)
	}

	go func(engine *nbio.Engine) {
		for {
			if idleConns.Load() >= MAX_IDLE_CONNS {
				continue
			}

			conn, err := nbio.Dial("tcp", taddr)
			if err != nil {
				log.Printf("err: %v\n", err)
				continue
			}

			engine.AddConn(conn)
			idleConns.Add(1)
			log.Printf("idle conns: %v\n", idleConns.Load())
		}
	}(tengine)

	return tengine, dengine
}
