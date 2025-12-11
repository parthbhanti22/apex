package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
	"math/rand"
)

// List of all potential leader ports
var ports = []string{"9001", "9002", "9003"}

func main() {
	fmt.Println("🔥 Starting Chaos Monkey Stress Test...")

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("user_%d", i)
		val := fmt.Sprintf("data_%d", rand.Intn(1000))
		
		success := false
		
		// Try to send to any random node (Client-side discovery)
		// In real life, we'd ask who the leader is, but here we just retry.
		for _, port := range ports {
			if sendSet(port, key, val) {
				fmt.Printf("✅ Set %s = %s (via :%s)\n", key, val, port)
				success = true
				break
			}
		}

		if !success {
			fmt.Printf("❌ Cluster Down! Retrying...\n")
			time.Sleep(1 * time.Second)
		}
		
		time.Sleep(50 * time.Millisecond) // Fast writes
	}
}

func sendSet(port, key, val string) bool {
	conn, err := net.DialTimeout("tcp", "localhost:"+port, 1*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	// 1. Command (SET = 1)
	conn.Write([]byte{0x01}) 
	
	// 2. Headers
	binary.Write(conn, binary.LittleEndian, uint32(len(key)))
	binary.Write(conn, binary.LittleEndian, uint32(len(val)))
	
	// 3. Body
	conn.Write([]byte(key))
	conn.Write([]byte(val))

	// 4. Read Ack
	ack := make([]byte, 1)
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, err = conn.Read(ack)
	if err != nil || ack[0] != 0x00 {
		return false
	}
	return true
}