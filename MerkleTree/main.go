package MerkleTree

import (
	"fmt"
	"i200547_BlockchainCrypto/Hashing"
)

type MerkleTree struct {
	Data  string
	left  *MerkleTree
	right *MerkleTree
}

func createNode(data string) *MerkleTree {
	return &MerkleTree{Data: Hashing.CalculateSha256(data)}
}

func CreateTree(data []string) *MerkleTree {
	nodes := make([]*MerkleTree, len(data))
	for i := range data {
		nodes[i] = createNode(data[i])
	}
	for len(nodes) > 1 {
		if len(nodes)%2 != 0 {
			nodes = append(nodes, nodes[len(nodes)-1])
		}
		temp := make([]*MerkleTree, 0)
		for i := 0; i < len(nodes); i += 2 {
			parent := createNode(Hashing.CalculateSha256(nodes[i].Data + nodes[i+1].Data))
			parent.left = nodes[i]
			parent.right = nodes[i+1]
			temp = append(temp, parent)
		}
		nodes = temp
	}
	return nodes[0]
}

func printTree(node *MerkleTree) {
	if node == nil {
		return
	}
	fmt.Printf("P: %s\n", node.Data)
	if node.left != nil {
		fmt.Printf("L: %s", node.left.Data)
	}
	if node.right != nil {
		fmt.Printf("R: %s\n", node.right.Data)
	} else {
		fmt.Println()
	}
	printTree(node.left)
	printTree(node.right)
}

func main() {
	data := []string{"Hello", "Yes", "I", "am", "inevitable"}
	root := CreateTree(data)
	printTree(root)
}
