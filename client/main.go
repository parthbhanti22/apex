package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// --- TEST SET ---
	fmt.Println("Sending SET command...")
	key := "hero"
	val := "Batman"

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
	conn.Read(ack)
	if ack[0] == 0x00 {
		fmt.Println("✅ SET Success")
	} else {
		fmt.Println("❌ SET Failed")
	}

	// --- TEST GET ---
	fmt.Println("Sending GET command...")
	
	// 1. Command (GET = 2)
	conn.Write([]byte{0x02})

	// 2. Header
	binary.Write(conn, binary.LittleEndian, uint32(len(key)))
	
	// 3. Body
	conn.Write([]byte(key))

	// 4. Read Response
	status := make([]byte, 1)
	conn.Read(status)
	if status[0] == 0x00 {
		var valLen uint32
		binary.Read(conn, binary.LittleEndian, &valLen)
		valBuf := make([]byte, valLen)
		conn.Read(valBuf)
		fmt.Printf("✅ GET Result: %s\n", string(valBuf))
	} else {
		fmt.Println("❌ GET Failed (Not Found)")
	}
}