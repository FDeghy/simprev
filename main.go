package main

import (
	"flag"
	"log"
	"net"
	"time"
)

func main() {
	mode := flag.Int("m", 0, "mode")
	flag.Parse()

	if *mode == 0 {
		addr, _ := net.ResolveIPAddr("ip:17", "127.0.0.1")
		conn, err := net.ListenIP("ip:17", addr)
		if err != nil {
			log.Fatalln(err)
		}
		data := make([]byte, 2048)
		log.Printf("%v, %v", conn.LocalAddr(), conn.RemoteAddr())
		for {
			n, a, err := conn.ReadFrom(data)
			log.Printf("%+v, %+v, %+v", a, err, string(data[:n]))
		}
	} else {
		addr, _ := net.ResolveIPAddr("ip:17", "127.0.0.1")
		conn, err := net.ListenIP("ip:17", addr)
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("%v, %v", conn.LocalAddr(), conn.RemoteAddr())
		for {
			n, err := conn.WriteTo([]byte("salam"), addr)
			log.Printf("%+v, %+v", err, n)
			time.Sleep(1 * time.Second)
		}
	}
}
