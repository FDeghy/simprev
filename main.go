package main

import (
	"flag"
	"log"
)

func main() {
	iran := flag.Bool("i", false, "iran")
	kharej := flag.Bool("f", false, "kharej")
	taddr := flag.String("la", "0.0.0.0:9985", "tunnel address (listen for iran, dial for kharej)")
	paddr := flag.String("pa", "127.0.0.1:51900", "proxy address")
	flag.Parse()

	if *iran {
		tEngine := StartTunnelServer(*taddr)
		defer tEngine.Stop()
		sEngine := StartServer(*paddr)
		defer sEngine.Stop()
	} else if *kharej {
		tEngine, dEngine := StartClientTunnel(*taddr, *paddr)
		defer tEngine.Stop()
		defer dEngine.Stop()
	} else {
		log.Fatalln("select i or f")
	}

	<-make(chan int)
}
