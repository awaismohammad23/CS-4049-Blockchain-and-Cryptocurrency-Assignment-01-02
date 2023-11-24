package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"
)

var flag = false

const networkType = "tcp"

type Message struct {
	Text string
}

type Node struct {
	IP         string
	ServerPort string
	ClientPort string
	message    Message
	Neighbors  []*Node
}

type BootstrapNode struct {
	IP       string
	Port     string
	Nodes    map[string]string // Map of node IP addresses to their server ports
	mutex    sync.Mutex
	waitlist chan *Node
}

func main() {
	var wg sync.WaitGroup

	// Create the bootstrap node
	bootstrap := &BootstrapNode{
		IP:       "127.0.0.1", // Change to the desired IP address of the bootstrap node
		Port:     "9090",      // Change to the desired port of the bootstrap node
		Nodes:    make(map[string]string),
		waitlist: make(chan *Node),
	}

	// Start the bootstrap node
	wg.Add(1)
	go startBootstrapNode(&wg, bootstrap)

	// Allow some time for the bootstrap node to start
	fmt.Println("Waiting for the bootstrap node to start...")
	fmt.Println("Press Enter to continue.")
	fmt.Scanln()

	// Create initial nodes
	nodes := []*Node{
		{IP: "127.0.0.1", ServerPort: "8080", ClientPort: "8081"},
		{IP: "127.0.0.1", ServerPort: "8082", ClientPort: "8083"},
	}

	// Start all nodes
	for _, node := range nodes {
		wg.Add(1)
		go startNode(&wg, node, nodes, bootstrap)
	}

	// Wait for all nodes to finish
	wg.Wait()
}
func getRandomNode(nodes []*Node, sourceNode *Node) *Node {
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(nodes))
	randomNode := nodes[randomIndex]

	// Ensure the selected node is not the source node
	for randomNode == sourceNode {
		randomIndex = rand.Intn(len(nodes))
		randomNode = nodes[randomIndex]
	}

	return randomNode
}
func startBootstrapNode(wg *sync.WaitGroup, bootstrap *BootstrapNode) {
	defer wg.Done()

	// Start the server for the bootstrap node
	bootstrapAddress := bootstrap.IP + ":" + bootstrap.Port
	ln, err := net.Listen(networkType, bootstrapAddress)
	if err != nil {
		fmt.Println("Error starting bootstrap server:", err)
		os.Exit(1)
	}
	defer ln.Close()

	fmt.Printf("Bootstrap Node: Server started and listening on %s\n", bootstrapAddress)

	// Handle incoming connections in a goroutine
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				fmt.Printf("Error accepting connection: %v\n", err)
				return
			}
			fmt.Printf("Bootstrap Node: Connection accepted from %s\n", conn.RemoteAddr())
			go handleBootstrapConnection(conn, bootstrap)
		}
	}()

	// Register nodes with the bootstrap node
	for {
		select {
		case node := <-bootstrap.waitlist:
			bootstrap.mutex.Lock()
			bootstrap.Nodes[node.IP] = node.ServerPort
			bootstrap.mutex.Unlock()
		}
	}
}

func handleBootstrapConnection(conn net.Conn, bootstrap *BootstrapNode) {
	defer conn.Close()

	decoder := gob.NewDecoder(conn)

	// Handle incoming registration messages from nodes
	for {
		var registrationMessage Message
		err := decoder.Decode(&registrationMessage)
		if err != nil {
			fmt.Println("Error decoding registration message:", err)
			return
		}

		// Assume the message contains the node's IP address
		nodeIP := registrationMessage.Text

		bootstrap.mutex.Lock()
		_, exists := bootstrap.Nodes[nodeIP]
		bootstrap.mutex.Unlock()

		// If the node is not already registered, add it to the waitlist
		if !exists {
			node := &Node{IP: nodeIP}
			bootstrap.waitlist <- node
		}
	}
}

func startNode(wg *sync.WaitGroup, node *Node, allNodes []*Node, bootstrap *BootstrapNode) {
	defer wg.Done()

	fmt.Printf("Node started at %s:%s (Server) and %s:%s (Client)\n", node.IP, node.ServerPort, node.IP, node.ClientPort)

	// Register the node with the bootstrap node
	registerWithBootstrap(node, bootstrap)

	// Start the server
	serverAddress := node.IP + ":" + node.ServerPort
	ln, err := net.Listen(networkType, serverAddress)
	if err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}
	defer ln.Close()

	fmt.Printf("Node: Server started and listening on %s\n", serverAddress)

	// Use a WaitGroup to wait for all server goroutines to finish
	var serverWG sync.WaitGroup
	serverWG.Add(1)

	// Handle incoming connections in a goroutine
	go func() {
		defer serverWG.Done()

		for {
			conn, err := ln.Accept()
			if err != nil {
				fmt.Printf("Error accepting connection: %v\n", err)
				return
			}
			fmt.Printf("Node %s: Connection accepted from %s\n", node.ServerPort, conn.RemoteAddr())

			// Start a goroutine to handle the connection
			serverWG.Add(1)
			go func() {
				defer serverWG.Done()
				handleConnection(conn)
			}()
		}
	}()

	// Start sending messages in a loop
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			message := fmt.Sprintf("Hello from %s", node.ClientPort)
			if flag == true {
				message = scanner.Text()

			}
			flag = true
			// Send a message to a random node in the P2P network
			randomNode := getRandomNode(allNodes, node)
			sendMessage(node, randomNode, message)
			sendMessage(randomNode, node, message)
			// Sleep for a while before sending the next message
			time.Sleep(5 * time.Second)
		}
	}()

	// Start the client
	for _, otherNode := range allNodes {
		if otherNode != node {
			go startClient(node, otherNode)
		}
	}

	// Wait for all server goroutines to finish before closing the listener
	serverWG.Wait()

	// Close the listener after all server goroutines have finished
	ln.Close()
}

func registerWithBootstrap(node *Node, bootstrap *BootstrapNode) {
	clientAddress := bootstrap.IP + ":" + bootstrap.Port
	conn, err := net.Dial(networkType, clientAddress)
	if err != nil {
		fmt.Printf("Error connecting from %s to bootstrap node %s:%s\n", node.ClientPort, bootstrap.IP, bootstrap.Port)
		return
	}
	defer conn.Close()

	fmt.Printf("Node %s: Connected to bootstrap node %s:%s\n", node.ClientPort, bootstrap.IP, bootstrap.Port)

	encoder := gob.NewEncoder(conn)

	// Send a registration message to the bootstrap node
	registrationMessage := Message{Text: node.IP}
	err = encoder.Encode(registrationMessage)
	if err != nil {
		fmt.Printf("Error encoding and sending registration message from %s to bootstrap node %s:%s\n", node.ClientPort, bootstrap.IP, bootstrap.Port)
		return
	}
}

func startClient(sourceNode *Node, targetNode *Node) {
	// Update the client port dynamically
	sourceNode.ClientPort = targetNode.ServerPort

	clientAddress := sourceNode.IP + ":" + sourceNode.ClientPort
	conn, err := net.Dial(networkType, clientAddress)
	if err != nil {
		fmt.Printf("Error connecting from %s to %s:%s\n", sourceNode.ClientPort, targetNode.IP, targetNode.ServerPort)
		return
	}
	defer conn.Close()

	fmt.Printf("Node %s: Connected to %s:%s\n", sourceNode.ClientPort, targetNode.IP, targetNode.ServerPort)

	encoder := gob.NewEncoder(conn)

	// Send a message from the client to the target node
	message := Message{Text: "Hello from " + sourceNode.ClientPort}
	err = encoder.Encode(message)
	if err != nil {
		fmt.Printf("Error encoding and sending message from %s to %s:%s\n", sourceNode.ClientPort, targetNode.IP, targetNode.ServerPort)
		return
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	decoder := gob.NewDecoder(conn)

	// Handle incoming messages from the connected peer
	for {
		var receivedMessage Message
		err := decoder.Decode(&receivedMessage)
		if err != nil {
			fmt.Println("Error decoding received message:", err)
			return
		}

		fmt.Printf("Received message from %s: %s\n", conn.RemoteAddr(), receivedMessage.Text)
	}
}

func sendMessage(sourceNode *Node, targetNode *Node, message string) {
	// Start a new goroutine for the client to send a message
	go func() {
		sourceNode.ClientPort = targetNode.ServerPort
		sourceNode.message.Text = message
		startClient(sourceNode, targetNode)
	}()
}
