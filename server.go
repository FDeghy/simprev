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

	engine.OnOpen(func(userConn *nbio.Conn) {
		tunnelConn := GetTunnelConn()
		if tunnelConn == nil {
			userConn.Close()
			log.Println("free tunnel connection NOT found")
			return
		}
		userConn.SetSession(tunnelConn)

		tunnelConn.OnData(func(tConn *nbio.Conn, data []byte) {
			_, err := userConn.Write(data)
			if err != nil {
				userConn.Close()
				tConn.Close()
				log.Printf("write tunnel->user error: %v\n", err)
				return
			}

			userConn.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
			tunnelConn.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
		})

		userConn.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
		tunnelConn.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
	})

	engine.OnData(func(userConn *nbio.Conn, data []byte) {
		tunnelConn, _ := userConn.Session().(*nbio.Conn)
		if tunnelConn == nil {
			userConn.Close()
			tunnelConn.Close()
			log.Println("OnData(user) tunnel connection NOT found")
			return
		}
		_, err := tunnelConn.Write(data)
		if err != nil {
			userConn.Close()
			tunnelConn.Close()
			log.Printf("write user->tunnel error: %v\n", err)
			return
		}

		userConn.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
		tunnelConn.SetReadDeadline(time.Now().Add(MAX_IDLE_TIME))
	})

	engine.OnClose(func(c *nbio.Conn, _ error) {
		tunnelConn, _ := c.Session().(*nbio.Conn)
		if tunnelConn != nil {
			tunnelConn.Close()
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

	engine.OnOpen(func(tConn *nbio.Conn) {
		err := PutTunnelConn(tConn)
		if err != nil {
			tConn.Close()
			log.Printf("failed to accept tunnel connection, err: %v\n", err)
		}
	})

	engine.OnData(func(tConn *nbio.Conn, data []byte) {
		tConn.DataHandler()(tConn, data)
	})

	engine.OnClose(func(tConn *nbio.Conn, _ error) {
		sess, _ := tConn.Session().(*nbio.Conn)
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
