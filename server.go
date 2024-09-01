package main

import (
	"log"
	"runtime"
	"time"

	"github.com/lesismal/nbio"
)

func StartServer(laddr string) *nbio.Engine {
	engine := nbio.NewEngine(nbio.Config{
		Network:            "tcp",
		Addrs:              []string{laddr},
		MaxWriteBufferSize: 6 * 1024 * 1024,
		NPoller:            runtime.NumCPU(),
	})

	engine.OnOpen(func(c *nbio.Conn) {
		sess := GetTunnelConn()
		if sess == nil {
			c.Close()
			log.Println("free tunnel connection NOT found")
			return
		}
		c.SetSession(sess)

		sess.OnData(func(conn *nbio.Conn, data []byte) {
			_, err := c.Write(data)
			if err != nil {
				c.Close()
				sess.Close()
				log.Printf("write tunnel->user error: %v\n", err)
				return
			}

			c.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
			sess.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
		})

		c.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
		sess.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
	})

	engine.OnData(func(c *nbio.Conn, data []byte) {
		sess, _ := c.Session().(*nbio.Conn)
		if sess == nil {
			c.Close()
			sess.Close()
			log.Println("OnData(user) tunnel connection NOT found")
			return
		}
		_, err := sess.Write(data)
		if err != nil {
			c.Close()
			sess.Close()
			log.Printf("write user->tunnel error: %v\n", err)
			return
		}

		c.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
		sess.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
	})

	engine.OnClose(func(c *nbio.Conn, _ error) {
		sess, _ := c.Session().(*nbio.Conn)
		if sess != nil {
			sess.Close()
		}
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
		NPoller:            runtime.NumCPU(),
	})

	engine.OnOpen(func(c *nbio.Conn) {
		err := PutTunnelConn(c)
		if err != nil {
			c.Close()
			log.Printf("failed to accept tunnel connection, err: %v\n", err)
		}
		log.Println("new tunnel connection accepted")
	})

	engine.OnData(func(c *nbio.Conn, data []byte) {
		c.DataHandler()(c, data)
	})

	engine.OnClose(func(c *nbio.Conn, _ error) {
		sess, _ := c.Session().(*nbio.Conn)
		if sess != nil {
			sess.Close()
		}
	})

	err := engine.Start()
	if err != nil {
		log.Fatalln(err)
	}

	return engine
}
