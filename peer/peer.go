package peer

import (
	"context"
	"errors"
	"log"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ahmed-debbech/p2p/model"
	"github.com/ahmed-debbech/p2p/protocol"
)

var (
	StunHost                     = "localhost:5000"
	FrameSize                    = 65547
	SyncIntervalMs time.Duration = 500 * time.Millisecond
	StunTimeout    time.Duration = 60 * time.Second
	StunKey                      = "1234"
	PeerWaitSeqNum time.Duration = 20 * time.Second
)

func StartPeer() {

	log.Println("Started as Peer")
	lifetimeSocket, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 0})
	if err != nil {
		log.Fatal("can not start unique socket .. shutting down", err)
	}
	defer lifetimeSocket.Close()

	peer, err := lookForPeer(lifetimeSocket)
	if err != nil {
		log.Fatal("SHUTTING DOWN...", err)
	}

	if peer == (model.Peer{}) {
		return
	}

	log.Println("Peer found: ", peer.IP, ":", peer.Port)

	sendCh := make(chan []byte)
	recvCh := make(chan []byte)

	ctx, cancel := context.WithCancel(context.Background())
	go setupConnection(ctx, cancel, lifetimeSocket, sendCh, recvCh, peer)
	DelegateToApp(ctx, sendCh, recvCh)
	<-ctx.Done()
}

func setupConnection(ctx context.Context, cancel context.CancelFunc, lifetimeSocket *net.UDPConn, sendCh chan []byte, recvCh chan []byte, peer model.Peer) {
	var nextSequenceNumber atomic.Uint64
	nextSequenceNumber.Store(1)

	log.Println("Trying to setup connection to", peer.IP, "on port", peer.Port)
	defer cancel()

	peerAddr, err := net.ResolveUDPAddr("udp4", peer.IP+":"+peer.Port)
	if err != nil {
		log.Println("Could not resolve peer host ", peer.IP, peer.Port, err)
		return
	}

	onSync := make(chan struct{}, 1)
	defer close(onSync)

	wg := new(sync.WaitGroup)
	wg.Add(2)

	var frameTimeout context.Context
	var cancelFrameTimeout context.CancelFunc
	defer func() {
		if cancelFrameTimeout != nil {
			cancelFrameTimeout()
		}
	}()

	go func() {
		defer close(recvCh)
		defer wg.Done()
		defer cancel()

		for {
			buffer := make([]byte, FrameSize)

			select {
			case <-ctx.Done():
				return
			default:
			}
			_, _, err := lifetimeSocket.ReadFromUDP(buffer)
			if err != nil {
				log.Printf("Error reading response from peer: %v", err)
				return
			}
			frame, err := protocol.ParseFrame(buffer)
			if err != nil {
				log.Println(err)
				return
			}

			seqRecv := frame.SequenceNumber
			log.Println("received seq number", seqRecv, "from peer")
			if cancelFrameTimeout != nil {
				cancelFrameTimeout()
			}
			if nextSequenceNumber.Load() != seqRecv {
				log.Println("error expected sequence number mismatch ", nextSequenceNumber.Load(), "!=", seqRecv, ".. break")
				// we should here break the connection immediately if it is not the expected sequence number
				return
			}
			log.Println("seq num matches with current", nextSequenceNumber.Load())

			select {
			case onSync <- struct{}{}:
			default:
			}

			if frame.PayloadLength == 0 {
				continue //that's a heartbeat
			}

			recvCh <- frame.Payload
		}
	}()

	go func() {
		defer close(sendCh)
		defer wg.Done()
		defer cancel()

		for {

			select {

			case <-ctx.Done():
				return
			case <-onSync:
				nextSequenceNumber.Add(1)
				if err := readFromSendChAndWriteUdp(sendCh, &nextSequenceNumber, lifetimeSocket, peerAddr); err != nil {
					return
				}
				frameTimeout, cancelFrameTimeout = context.WithTimeout(context.Background(), PeerWaitSeqNum)
				<-frameTimeout.Done() // wait for a response from peer
				if frameTimeout.Err() == context.DeadlineExceeded {
					//we waited long enough for peer to respond but yet they didn't
					log.Println("didn't receive any new frames from peer after", PeerWaitSeqNum.String(), "timeout")
					return
				}
			default:
				if nextSequenceNumber.Load() == 1 {

					select {
					case <-ctx.Done():
						return
					default:
						if err := readFromSendChAndWriteUdp(sendCh, &nextSequenceNumber, lifetimeSocket, peerAddr); err != nil {
							return
						}
						time.Sleep(SyncIntervalMs)
					}
					continue
				}
			}
		}
	}()

	wg.Wait()

}

func readFromSendChAndWriteUdp(sendCh chan []byte, nextSequenceNumber *atomic.Uint64,
	lifetimeSocket *net.UDPConn, peerAddr *net.UDPAddr) error {

	var toSendBytes []byte = make([]byte, 0)
	select {
	case data, ok := <-sendCh:
		if !ok {
			return errors.New("sendCh is closed")
		}
		toSendBytes = data
	default:
	}

	binaryFrame, err := protocol.BuildFrameToBinary(nextSequenceNumber.Load(), toSendBytes)
	if err != nil {
		log.Println(err)
		return nil
	}
	log.Println("sending seq", nextSequenceNumber.Load(), "to peer")
	_, err = lifetimeSocket.WriteToUDP(binaryFrame, peerAddr)
	if err != nil {
		log.Printf("Error sending message from peer: %v", err)
		return errors.New("Error sending message from peer")
	}
	return nil
}

func lookForPeer(lifetimeSocket *net.UDPConn) (model.Peer, error) {
	tcpServer, err := net.ResolveUDPAddr("udp4", StunHost)
	if err != nil {
		log.Println("could not resolve STUN hostname", err.Error())
		return model.Peer{}, errors.New("could not resolve STUN hostname " + err.Error())
	}

	// Send message to server
	_, err = lifetimeSocket.WriteToUDP(append([]byte(StunKey), strconv.Itoa(int(StunTimeout.Seconds()))...), tcpServer)
	if err != nil {
		log.Println("could write to initiate the connection to STUN", err.Error())
		return model.Peer{}, errors.New("could write to initiate the connection to STUN " + err.Error())
	}
	n := 0
	var readError error = nil
	received := make([]byte, 4096)
	interrupt := make(chan struct{})
	go func() {
		n, _, err = lifetimeSocket.ReadFromUDP(received)
		interrupt <- struct{}{}
		readError = err
		if err != nil {
			log.Println("could not read peers from STUN server", err.Error())
		}
	}()

	ticker := time.NewTicker(StunTimeout)
	defer ticker.Stop()
	select {
	case <-ticker.C:
		log.Println("no peers have been found on stun with that key")
		return model.Peer{}, nil
	case <-interrupt:
		ticker.Stop()
		break
	}

	if readError != nil {
		return model.Peer{}, errors.New("could not read peers from STUN server " + readError.Error())
	}

	peersInBinary := received[:n]

	peers := make([]model.Peer, 0)
	i := 0
	j := 0
	for i <= n-1 {
		if peersInBinary[i] == '#' {
			pp := model.Peer{}
			pBin := peersInBinary[j:i]
			n := 0
			for k := 1; k <= 3; k++ {
				st := ""
				for pBin[n] != ',' {
					st += string(pBin[n])
					n++
				}
				if k == 1 {
					pp.ID = st
				}
				if k == 2 {
					pp.IP = st
				}
				if k == 3 {
					pp.Port = st
				}
				n++
			}
			peers = append(peers, pp)

			j = i + 1
		}
		i++

	}
	return peers[0], nil
}
