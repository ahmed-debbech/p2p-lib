package main

import (
	"log"
	"os"

	"github.com/ahmed-debbech/p2p-lib/peer"
	"github.com/ahmed-debbech/p2p-lib/stun"
)

func main() {
	log.Println("P2P - Hello World")

	if len(os.Args) != 2 {
		log.Fatal("wrong number of arguments")
	}

	if os.Args[1] == "stun" {
		log.Println("Running as STUN")
		stun.StartStun()
	}
	if os.Args[1] == "peer" {
		log.Println("Running as peer")
		peer.StartPeer()
	}
}
