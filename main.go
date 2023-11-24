package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

type Transaction struct {
	transaction    string
	added_to_block bool
}

type Peer struct {
	IP   string
	Port string
}

var bootstrapNode = Peer{IP: "127.0.0.1", Port: "8000"}
var peers []Peer
var connections = make(map[net.Conn]bool)
var mutex = &sync.Mutex{}
var Ledger = make(map[string][]Transaction)
var transactions = []Transaction{}

// hello
func main() {
	var wg sync.WaitGroup

	// Start the bootstrap node
	wg.Add(1)
	go startServer(bootstrapNode, &wg)

	wg.Wait()
}

func startServer(peer Peer, wg *sync.WaitGroup) {
	defer wg.Done()

	ln, _ := net.Listen("tcp", net.JoinHostPort(peer.IP, peer.Port))
	defer ln.Close()

	for {
		conn, _ := ln.Accept()
		mutex.Lock()
		connections[conn] = true
		mutex.Unlock()
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		peerAddress := conn.RemoteAddr().String()
		if err != nil {
			// If there's an error, it means there's no more data to read from the connection
			break
		}
		var PeerExist = false
		for i := range peers {
			if peers[i].IP == strings.Split(peerAddress, ":")[0] &&
				peers[i].Port == strings.Split(peerAddress, ":")[1] {
				PeerExist = true
				break
			}
		}
		if PeerExist == false {
			// If a new peer is registering, add it to the list of peers
			fmt.Println("Registering.......")
			mutex.Lock()
			peers = append(peers, Peer{IP: strings.Split(peerAddress, ":")[0], Port: strings.Split(peerAddress, ":")[1]})
			mutex.Unlock()
			fmt.Println("Peer ", peerAddress, " has been registered.")
		}
		// Only print out the message if it's not empty
		if strings.TrimSpace(message) != "" {
			var ValidTransaction = true
			for i := range transactions {
				if transactions[i].transaction == message {
					ValidTransaction = false
					break
				}
			}
			fmt.Println("Message from", conn.RemoteAddr().String(), ":", message)

			{
				// Otherwise, broadcast the message to all other clients
				mutex.Lock()
				for clientConn := range connections {

					if ValidTransaction == true {
						transactions = append(transactions, Transaction{transaction: message, added_to_block: false})
						Ledger[clientConn.LocalAddr().String()] = transactions
						//fmt.Println(clientConn.RemoteAddr().String())
					}
					if clientConn != conn && ValidTransaction == true {
						fmt.Fprint(clientConn, message)
					} else if clientConn == conn && ValidTransaction == false {
						fmt.Fprint(clientConn, "Transaction not valid.")
					}
				}
				mutex.Unlock()
			}
		}
	}
}

//Bootstrap Node added
//Peer2Peer added
//Create P2P network task completed
