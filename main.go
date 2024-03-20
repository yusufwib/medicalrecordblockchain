package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Block represents a block in the blockchain
type Block struct {
	ID        string `json:"id"`
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
		ID:        uuid.New().String(),
		Timestamp: "2024-03-21 00:00:00",
		Data:      "Genesis Block",
		PrevHash:  "",
		Hash:      calculateHash("genesisHash", "2024-03-21 00:00:00", "Genesis Block", ""),
	}
	blockchain.Chain = append(blockchain.Chain, genesisBlock)
}

func mineBlock(nodeID string) Block {
	// In a real implementation, you would perform proof-of-work here
	// For simplicity, let's just create a new block with arbitrary data
	prevBlock := blockchain.Chain[len(blockchain.Chain)-1]
	newBlock := Block{
		ID:        uuid.New().String(),
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Data:      fmt.Sprintf("Data for Node %s", nodeID),
		PrevHash:  prevBlock.Hash,
		Hash:      calculateHash(prevBlock.Hash, time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf("Data for Node %s", nodeID), prevBlock.Hash),
	}
	blockchain.Chain = append(blockchain.Chain, newBlock)

	storeBlockData(newBlock, nodeID)

	return newBlock
}

func calculateHash(prevHash, timestamp, data, prevHash2 string) string {
	blockData := fmt.Sprintf("%s%s%s%s", prevHash, timestamp, data, prevHash2)
	hash := sha256.Sum256([]byte(blockData))
	return fmt.Sprintf("%x", hash)
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

	// Remove any block with the same ID
	updatedBlocks := make([]Block, 0)
	for _, b := range blocks {
		if b.ID != block.ID {
			updatedBlocks = append(updatedBlocks, b)
		}
	}

	// Append the new block
	updatedBlocks = append(updatedBlocks, block)

	// Marshal the updated blocks
	data, err = json.Marshal(updatedBlocks)
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
	// TODO: sync only not crate a new block
	for {
		// Simulate mining process (for demonstration purpose, it's just sleeping for a few seconds)
		time.Sleep(10 * time.Second)

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
	// Query discovery service to get other nodes
	otherNodes, err := loadNodes()
	if err != nil {
		return err
	}

	// Iterate over other nodes and update block data files
	for _, node := range otherNodes {
		storeBlockData(newBlock, node)
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

func startHTTPServer() {
	e := echo.New()

	e.GET("/chain", func(c echo.Context) error {
		// Read blockchain data from files in the data directory
		blockchainData, err := readBlockchainData()
		if err != nil {
			log.Printf("Error reading blockchain data: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read blockchain data"})
		}
		return c.JSON(http.StatusOK, blockchainData)
	})

	// TODO: make the hash is correlated with the patient
	// store the block only for nodes primary first (optional)
	e.POST("/block", func(c echo.Context) error {
		var block Block
		if err := c.Bind(&block); err != nil {
			return err
		}

		// Mine a new block
		newBlock := mineBlock(block.Data)

		// Sync the new block with other nodes
		err := syncBlock(newBlock)
		if err != nil {
			log.Printf("Error syncing block: %v", err)
		}

		return c.JSON(http.StatusCreated, newBlock)
	})

	// Start HTTP server
	port := ":8080" // You can specify any port you want
	log.Printf("HTTP server listening on port %s...\n", port)
	e.Logger.Fatal(e.Start(port))
}

func readBlockchainData() ([]Block, error) {
	// Read blockchain data from files
	data, err := ioutil.ReadFile(dataDirectory + getPrimaryNodes() + ".json")
	if err != nil {
		return nil, err
	}

	var blocks []Block
	if err := json.Unmarshal(data, &blocks); err != nil {
		return nil, err
	}

	return blocks, nil
}

func getPrimaryNodes() string {
	nodes, err := loadNodes()
	if err != nil {
		log.Fatalf("Error loading nodes: %v", err)
	}
	return nodes[len(nodes)-1]
}
