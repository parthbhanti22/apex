package main

import "log"

func main() {
	db, err := NewDB("apex.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Start the server
	server := NewServer(db)
	server.Start(":8080")
}