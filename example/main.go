package main

import (
	"log"

	"github.com/t-ham752/go-mini-psqld/pkg/server"
)

const serverVersion = "14.11 (Debian 14.11-1.pgdg110+2)"

func main() {
	conf := &server.TCPServerConfig{
		Port:         54322,
		QueryHandler: queryHandler,
	}
	server := server.NewTCPServer(conf,
		server.WithServerVersion(serverVersion),
		server.WithTimeZone("Asia/Tokyo"),
	)

	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("failed to start server: %+v", err)
		}
	}()

	quit := make(chan struct{})
	<-quit
}

func queryHandler(query []byte) ([]byte, error) {
	// you can parse query and return response
	return []byte("OK!!!"), nil
}
