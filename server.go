package main

import (
	"log"
	"time"

	"github.com/lesismal/nbio"
)

func StartServer(laddr string) *nbio.Engine {
	engine := nbio.NewEngine(nbio.Config{
		Network:            "tcp",
		Addrs:              []string{laddr},
		MaxWriteBufferSize: 6 * 1024 * 1024,
	})

	engine.OnOpen(func(c *nbio.Conn) {
		sess := GetTunnelConn()
		if sess == nil {
			c.Close()
			return
		}
		c.SetSession(sess)

		sess.OnData(func(conn *nbio.Conn, data []byte) {
			_, err := c.Write(data)
			if err != nil {
				c.Close()
				sess.Close()
				return
			}

			c.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
			sess.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
		})

		c.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
		sess.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
	})

	engine.OnData(func(c *nbio.Conn, data []byte) {
		sess := c.Session().(*nbio.Conn)
		_, err := sess.Write(data)
		if err != nil {
			c.Close()
			sess.Close()
			return
		}

		c.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
		sess.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
	})

	engine.OnClose(func(c *nbio.Conn, _ error) {
		sess := c.Session().(*nbio.Conn)
		sess.Close()
	})

	err := engine.Start()
	if err != nil {
		log.Fatalln(err)
	}

	return engine
}

func StartTunnelServer(laddr string) *nbio.Engine {
	engine := nbio.NewEngine(nbio.Config{
		Network:            "tcp",
		Addrs:              []string{laddr},
		MaxWriteBufferSize: 6 * 1024 * 1024,
	})

	engine.OnOpen(func(c *nbio.Conn) {
		PutTunnelConn(c)
	})

	engine.OnData(func(c *nbio.Conn, data []byte) {
		c.DataHandler()(c, data)
	})

	err := engine.Start()
	if err != nil {
		log.Fatalln(err)
	}

	return engine
}
