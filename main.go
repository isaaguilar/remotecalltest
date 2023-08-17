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
	xterm "golang.org/x/term"
)

func main() {

	// Get the file descriptor of the terminal
	fd := int(os.Stdout.Fd())

	// Create a channel to receive terminal size changes
	sizeCh := make(chan ([2]int))
	columns, rows, _ := xterm.GetSize(fd)

	// Create a signal handler for SIGWINCH
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)

	// Run a goroutine to listen for signals and send the new size to the channel
	go func() {
		sizeCh <- [2]int{columns, rows}
		for range sigCh {
			columns, rows, err := xterm.GetSize(fd)
			if err == nil {
				sizeCh <- [2]int{columns, rows}
			}
		}
	}()

	if TerminalWebsocket(sizeCh) {

		log.Println("Done")
		return
	}

}

func TerminalWebsocket(sizeCh chan [2]int) (isDone bool) {
	// isKeyboardSet := false

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
					// log.Println("PONG!")
					continue
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

	// if err := keyboard.Open(); err != nil {
	// 	panic(err)
	// }

	keyEvent, err := keyboard.GetKeys(1)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	var ctrlCExit bool
	for {

		select {
		case size := <-sizeCh:

			sizeJSON := fmt.Sprintf(`{"Columns": %d, "Rows": %d}`, size[0], size[1])
			encodedSizeJSON := b64.StdEncoding.EncodeToString([]byte(sizeJSON))
			// log.Println("Will set size", sizeJSON)
			err := conn.WriteMessage(websocket.TextMessage, append([]byte{byte('3')}, []byte(encodedSizeJSON)...))
			// err := conn.WriteMessage(websocket.TextMessage, []byte{byte('2')})
			if err != nil {
				log.Println("there was some error:", err)
			}
			continue

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

		case ev := <-keyEvent:

			if ctrlCExit {
				fmt.Fprintf(os.Stderr, "\nPress ctrl-c again to exit session\n")
			}
			// Scan for input from the standard input
			// if !scanner.Scan() {
			// 	return
			// }
			// // Get the input text
			// text := scanner.Text()
			// encodedText := b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s\n", text)))
			// encodedText = "1" + encodedText
			// log.Println("Will send", encodedText)
			char, key, err := ev.Rune, ev.Key, ev.Err
			if err != nil {
				panic(err)
			}

			if key == keyboard.KeyCtrlC {
				if ctrlCExit {
					err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
					if err != nil {
						log.Println("write close:", err)
						return
					}
				}
				ctrlCExit = false
			} else {
				ctrlCExit = false
			}

			// fmt.Printf("%c", char)

			// var b byte
			var byteArr []byte
			if key != 0 {
				switch key {
				case keyboard.KeyArrowUp:
					byteArr = []byte{27, 91, 65}
				case keyboard.KeyArrowDown:
					byteArr = []byte{27, 91, 66}
				case keyboard.KeyArrowLeft:
					byteArr = []byte{27, 91, 68}
				case keyboard.KeyArrowRight:
					byteArr = []byte{27, 91, 67}
				default:
					byteArr = []byte{byte(key)}
				}

			} else {
				byteArr = []byte{byte(char)}
			}

			// log.Println("I pressed", char, "key", key, "byte", byteArr, "string", string(byteArr))

			encodedText := b64.StdEncoding.EncodeToString(byteArr)

			if key == keyboard.KeyHome {
				log.Println("I pressed the home key")
				err := conn.WriteMessage(websocket.TextMessage, []byte{byte('2')})
				if err != nil {
					log.Println(err)
				}
				continue
			}

			wstext := "1" + encodedText

			// Write a text message to the WebSocket connection
			erre := conn.WriteMessage(websocket.TextMessage, []byte(wstext))
			if erre != nil {
				log.Println("write:", erre)
				return
			}

		}
	}

}
