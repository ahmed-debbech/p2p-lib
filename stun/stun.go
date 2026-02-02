package stun

import (
	"log"
	"net"
	"strconv"
	"time"

	"github.com/ahmed-debbech/p2p-lib/model"
	"github.com/google/uuid"
)

var (
	SERVER_PORT = "5000"
)

type Room struct {
	PeerA       *model.Peer
	PeerB       *model.Peer
	PeerAClient *net.UDPAddr
	PeerBClient *net.UDPAddr
}

var (
	RoomKeyMapping map[string]Room = make(map[string]Room)
)

func StartStun() {

	addr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:"+SERVER_PORT)
	if err != nil {
		log.Fatal("Error resolving UDP address in running STUN", err)
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Fatal("Error listening on UDP in running STUN", err)
	}

	buffer := make([]byte, 1024)

	log.Print("STUN server is up&running")

	go AlwaysFullfillOrDeleteRooms(RoomKeyMapping, conn)

	for {
		// Read incoming message
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading UDP message: %v", err)
			continue
		}

		if len(buffer[:n]) != 6 {
			log.Println(len(buffer), "request error: as we say in HTTP 400 BAD REQUEST")
			continue
		}

		key := string(buffer[:4])
		timeoutSec := string(buffer[4:6])
		timeoutSecInt, err := strconv.Atoi(timeoutSec)
		if err != nil {
			log.Println("could not figure out how long to time out for peer with key", key)
			continue
		}
		log.Println("recevied peer on STUN", clientAddr, "with key", key, "and timeout of", timeoutSec)

		if room, ok := RoomKeyMapping[key]; ok { //room already exist
			// add peerB second
			PeerBClient := clientAddr
			PeerB := &model.Peer{
				ID:            uuid.New().String(),
				IP:            string(clientAddr.IP.String()),
				Port:          strconv.Itoa(clientAddr.Port),
				FirstAppeared: time.Now().Unix(),
				/*
				* TimeoutInSec: --
				* here as you see the timeout for peerB is always ignored
				* because the room always start the countdown based on the first peer
				* to create the room which is always peerA
				* and because we do not know who will come first peerA/B we let both
				* peers send the STUN a timeout, but second one is always ignored
				 */
			}
			room.PeerB = PeerB
			room.PeerBClient = PeerBClient
			RoomKeyMapping[key] = room

		} else { //room does not exist add peerA first
			PeerAClient := clientAddr
			PeerA := &model.Peer{
				ID:            uuid.New().String(),
				IP:            string(clientAddr.IP.String()),
				Port:          strconv.Itoa(clientAddr.Port),
				TimeoutInSec:  timeoutSecInt,
				FirstAppeared: time.Now().Unix(),
			}

			room := Room{
				PeerA:       PeerA,
				PeerB:       nil,
				PeerAClient: PeerAClient,
				PeerBClient: nil,
			}
			RoomKeyMapping[key] = room
		}

	}

}

func AlwaysFullfillOrDeleteRooms(RoomKeyMapping map[string]Room, conn *net.UDPConn) {
	for {
		for k, v := range RoomKeyMapping {
			if (v.PeerA != nil) && (v.PeerB != nil) {

				// send peer b to a
				peersInfoBinary := make([]byte, 0)
				elementBin := make([]byte, 0)
				elementBin = append(elementBin, []byte(v.PeerB.ID)...)
				elementBin = append(elementBin, []byte{','}...)
				elementBin = append(elementBin, []byte(v.PeerB.IP)...)
				elementBin = append(elementBin, []byte{','}...)
				elementBin = append(elementBin, []byte(v.PeerB.Port)...)
				elementBin = append(elementBin, []byte{','}...)
				elementBin = append(elementBin, []byte{'#'}...)
				peersInfoBinary = append(peersInfoBinary, elementBin...)

				log.Println("ROOM[", k, "] replying to A", string(peersInfoBinary))
				_, err := conn.WriteToUDP(peersInfoBinary, v.PeerAClient)
				if err != nil {
					log.Printf("Error sending stun response to peer A: %v", err)
				}

				//send peer a to b
				peersInfoBinary = make([]byte, 0)
				elementBin = make([]byte, 0)
				elementBin = append(elementBin, []byte(v.PeerA.ID)...)
				elementBin = append(elementBin, []byte{','}...)
				elementBin = append(elementBin, []byte(v.PeerA.IP)...)
				elementBin = append(elementBin, []byte{','}...)
				elementBin = append(elementBin, []byte(v.PeerA.Port)...)
				elementBin = append(elementBin, []byte{','}...)
				elementBin = append(elementBin, []byte{'#'}...)
				peersInfoBinary = append(peersInfoBinary, elementBin...)

				log.Println("ROOM[", k, "]replying to B", string(peersInfoBinary))

				_, err = conn.WriteToUDP(peersInfoBinary, v.PeerBClient)
				if err != nil {
					log.Printf("Error sending stun response to peer B: %v", err)
				}
				delete(RoomKeyMapping, k)
			}
			if (int64(v.PeerA.TimeoutInSec) + v.PeerA.FirstAppeared) < time.Now().Unix() {
				//eligible to delete the room mapping
				delete(RoomKeyMapping, k)
				log.Println("deleted", k, "room mapping after", v.PeerA.TimeoutInSec, "peerB did come online")
			}
		}
		time.Sleep(time.Millisecond * 500)

	}
}
