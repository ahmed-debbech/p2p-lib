package peer

import (
	"bufio"
	"context"
	"log"
	"os"
)

func DelegateToApp(ctx context.Context, sendCh chan []byte, recvCh chan []byte) {
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		text := ""
		for scanner.Scan() {
			text = scanner.Text()
			if err := scanner.Err(); err != nil {
				log.Println("can't read your types from keyboard")
				continue
			}
			select {
			case sendCh <- []byte(text):
			case <-ctx.Done():
				log.Println("Channel is closed to send because underlying socket is closed")
				return
			}
		}
	}()
	go func() {

		for {
			receivedData, ok := <-recvCh
			if !ok {
				log.Println("Channel is closed to recive because underlying socket is closed")
				return
			}
			log.Println("Peer: ", string(receivedData))
		}
	}()
}
