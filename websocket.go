package client

import (
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/HackerLoop/rotonde/shared"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
)

func startConnection(rotondeUrl string, inChan, outChan chan interface{}) {
	log.Println("startRotondeClient")
	u, err := url.Parse(rotondeUrl)
	if err != nil {
		panic(err)
	}

	for {
		conn, err := net.Dial("tcp", u.Host)
		if err != nil {
			log.Println(err)
			time.Sleep(2 * time.Second)
			continue
		}
		ws, response, err := websocket.NewClient(conn, u, http.Header{}, 10000, 10000)
		if err != nil {
			log.Println(err)
			log.Println(response)
			time.Sleep(2 * time.Second)
			continue
		}
		processRotondePackets(ws, inChan, outChan)
	}
}

func processRotondePackets(conn *websocket.Conn, inChan, outChan chan interface{}) {
	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case dispatcherPacket := <-inChan:
				jsonPacket, err := rotonde.ToJSON(dispatcherPacket)
				if err != nil {
					log.Warning(err)
				}
				if err := conn.WriteMessage(websocket.TextMessage, jsonPacket); err != nil {
					log.Warning(err)
					return
				}

			case <-errChan:
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			messageType, reader, err := conn.NextReader()
			if err != nil {
				log.Println(err)
				errChan <- err
				return
			}
			if messageType == websocket.TextMessage {
				dispatcherPacket, err := rotonde.FromJSON(reader)
				if err != nil {
					log.Warning(err)
				}
				outChan <- dispatcherPacket
			}
		}
	}()

	log.Println("Treating messages")
	wg.Wait()
}
