package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
)

// Block represents a block in the blockchain
type Block struct {
	Index     int    `json:"index"`
	Timestamp string `json:"timestamp"`
	Data      string `json:"data"`
	PrevHash  string `json:"prevHash"`
	Hash      string `json:"hash"`
}

// Blockchain represents the blockchain
type Blockchain struct {
	Chain []Block `json:"chain"`
}

var blockchain Blockchain

// Data directory to store node data
const dataDirectory = "./data/"
const nodesFile = "nodes.json"

func main() {
	nodeID := flag.String("NODE_ID", "", "Node ID")
	httpMode := flag.Bool("HTTP", false, "HTTP mode")

	flag.Parse()

	if *httpMode {
		startHTTPServer()
	} else {
		if *nodeID == "" {
			fmt.Println("Please specify a node ID using --NODE_ID flag")
			os.Exit(1)
		}
		// Initialize blockchain
		initBlockchain()
		// Run the blockchain node
		runNode(*nodeID)
	}
}

func initBlockchain() {
	// Initialize blockchain with genesis block
	genesisBlock := Block{
		Index:     0,
		Timestamp: "2024-03-21 00:00:00",
		Data:      "Genesis Block",
		PrevHash:  "",
		Hash:      "genesisHash",
	}
	blockchain.Chain = append(blockchain.Chain, genesisBlock)
}

func mineBlock() Block {
	// In a real implementation, you would perform proof-of-work here
	// For simplicity, let's just create a new block with arbitrary data
	newBlock := Block{
		Index:     len(blockchain.Chain),
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Data:      fmt.Sprintf("Data for Node %s", nodeID),
		PrevHash:  blockchain.Chain[len(blockchain.Chain)-1].Hash,
		Hash:      "newBlockHash", // In real scenario, this should be calculated based on the block's data and other parameters
	}
	blockchain.Chain = append(blockchain.Chain, newBlock)

	otherNodes, _ := loadNodes()
	for _, nodeID := range otherNodes {
		storeBlockData(newBlock, nodeID)
	}

	return newBlock
}

func storeBlockData(block Block, nodeID string) {
	// Filename for storing block data
	filename := dataDirectory + fmt.Sprintf("%s.json", nodeID)

	// Read existing block data from file
	var blocks []Block
	data, err := ioutil.ReadFile(filename)
	if err == nil {
		err = json.Unmarshal(data, &blocks)
		if err != nil {
			log.Printf("Error unmarshalling block data: %v\n", err)
			return
		}
	} else if !os.IsNotExist(err) {
		log.Printf("Error reading existing block data: %v\n", err)
	}

	// Append the new block
	blocks = append(blocks, block)

	// Marshal the updated blocks
	data, err = json.Marshal(blocks)
	if err != nil {
		log.Printf("Error marshalling block data: %v\n", err)
		return
	}

	// Write updated data back to the file
	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		log.Printf("Error writing block data to file: %v\n", err)
	}
}

func runNode(nodeID string) {
	// Register with discovery service
	err := registerNodeWithDiscovery(nodeID)
	if err != nil {
		log.Fatalf("Error registering with discovery service: %v", err)
	}

	// Run the node logic here
	log.Printf("Node %s is running...\n", nodeID)

	// Simulate continuous mining and syncing
	for {
		// Simulate mining process (for demonstration purpose, it's just sleeping for a few seconds)
		time.Sleep(5 * time.Second)

		// Mine a new block
		newBlock := mineBlock(nodeID)

		// Sync the new block with other nodes
		err := syncBlock(newBlock)
		if err != nil {
			log.Printf("Error syncing block: %v", err)
		}
	}
}

func syncBlock(newBlock Block) error {
	// Serialize the new block
	blockBytes, err := json.Marshal(newBlock)
	if err != nil {
		return err
	}

	// Query discovery service to get other nodes
	otherNodes, err := loadNodes()
	if err != nil {
		return err
	}

	// Iterate over other nodes and send the new block to them
	for _, node := range otherNodes {
		url := fmt.Sprintf("http://%s:8080/sync", node) // Assuming /sync endpoint to receive new blocks
		_, err := http.Post(url, "application/json", bytes.NewBuffer(blockBytes))
		if err != nil {
			log.Printf("Error syncing with node %s: %v", node, err)
		}
	}

	return nil
}

func loadNodes() ([]string, error) {
	// Load nodes from nodes.json file
	fileData, err := ioutil.ReadFile(nodesFile)
	if os.IsNotExist(err) {
		// If file not found, create it
		err := createNodesFile()
		if err != nil {
			return nil, err
		}
		// Return an empty array since the file was just created
		return []string{}, nil
	} else if err != nil {
		return nil, err
	}

	var nodes []string
	if err := json.Unmarshal(fileData, &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func createNodesFile() error {
	// Create an empty nodes.json file
	emptyNodes := []string{}
	data, err := json.Marshal(emptyNodes)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(nodesFile, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func saveNodes(nodes []string) error {
	// Save nodes to nodes.json file
	data, err := json.Marshal(nodes)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(nodesFile, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func registerNodeWithDiscovery(nodeID string) error {
	// Load existing nodes
	nodes, err := loadNodes()
	if err != nil {
		return err
	}

	// Add new node
	nodes = append(nodes, nodeID)

	// Save updated nodes
	err = saveNodes(nodes)
	if err != nil {
		return err
	}

	return nil
}

func initNodeData() {
	// Initial nodes array
	nodes := []string{}

	// Marshal the nodes array to JSON
	data, err := json.Marshal(nodes)
	if err != nil {
		log.Fatalf("Error marshalling nodes array: %v", err)
	}

	// Write JSON data to nodes.json file
	err = ioutil.WriteFile("nodes.json", data, 0644)
	if err != nil {
		log.Fatalf("Error writing nodes.json file: %v", err)
	}

	log.Println("nodes.json file created successfully.")
}

func startHTTPServer() {
	e := echo.New()

	// GET /chain endpoint to get the blockchain
	e.GET("/chain", func(c echo.Context) error {
		return c.JSON(http.StatusOK, blockchain)
	})

	// POST /mine endpoint to mine a new block
	e.POST("/mine", func(c echo.Context) error {
		// Implement mining logic here
		newBlock := mineBlock()
		return c.JSON(http.StatusOK, newBlock)
	})

	// Start HTTP server
	port := ":8080" // You can specify any port you want
	log.Printf("HTTP server listening on port %s...\n", port)
	e.Logger.Fatal(e.Start(port))
}
