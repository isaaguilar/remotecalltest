package main

import (
	b64 "encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/gorilla/websocket"
)

func main() {
	if TerminalWebsocket() {

		log.Println("Done")
		return
	}

}

func TerminalWebsocket() (isDone bool) {
	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	isDone = true

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial("ws://localhost:5555/ws/debug/microk8s/default/hello-tfo", nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	// Create a channel for interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Create a channel for done signal
	done := make(chan struct{})

	// Start a goroutine to read messages from the WebSocket connection
	go func() {
		defer close(done)
		for {
			// Read a message
			mt, bmsg, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			// fmt.Printf("Received: %s\n", bmsg)
			// digit1, err := b64.StdEncoding.DecodeString(string(bmsg[1]))
			// if err != nil {
			// 	log.Println(err)
			// }
			// fmt.Printf("Received: %s\n", string(digit1))
			if mt == websocket.TextMessage {
				msg := string(bmsg)
				if strings.HasPrefix(msg, "2") {
					log.Println("Sending 2")
					conn.WriteMessage(1, bmsg)
				}
				if strings.HasPrefix(msg, "3") {
					continue
				}
				if strings.HasPrefix(msg, "6") {
					continue
				}

				dec, err := b64.StdEncoding.DecodeString(string(msg[1:]))
				if err != nil {
					log.Println(err)
					continue
				}
				fmt.Printf("%s", string(dec))

			} else {

				fmt.Printf("The MessageType: %+v\n", mt)
				// Print the message to the standard output
				fmt.Printf("Received: %s\n", bmsg)
			}
		}
	}()

	// Create a scanner to read from the standard input
	// scanner := bufio.NewScanner(os.Stdin)

	// Loop until done or interrupted
	for {
		select {
		case <-done:
			return
		case <-interrupt:
			// Send a close message to the WebSocket server
			log.Println("interrupt")
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			// Wait for the server to close the connection
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		default:
			// Scan for input from the standard input
			// if !scanner.Scan() {
			// 	return
			// }
			// // Get the input text
			// text := scanner.Text()
			// encodedText := b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s\n", text)))
			// encodedText = "1" + encodedText
			// log.Println("Will send", encodedText)
			char, key, err := keyboard.GetKey()
			if err != nil {
				panic(err)
			}

			// fmt.Printf("%c", char)

			b := []byte{}
			if key != 0 {
				b = append(b, byte(key))
			} else {
				b = append(b, []byte(string(char))...)
			}
			encodedText := b64.StdEncoding.EncodeToString(b)

			wstext := "1" + encodedText

			// Write a text message to the WebSocket connection
			erre := conn.WriteMessage(websocket.TextMessage, []byte(wstext))
			if erre != nil {
				log.Println("write:", erre)
				return
			}
			if key == keyboard.KeyCtrlC {
				return
			}
		}
	}

}
