package main

import (
	"flag"
	"fmt"
	"log"
	"mcp-bench/internal/transport"
)

func main() {
	tr := flag.String("transport", "stdio", "Transport: stdio or http")
	port := flag.Int("port", 8080, "Port for HTTP transport")
	flag.Parse()

	switch *tr {
	case "stdio":
		if err := transport.RunStdio(); err != nil {
			log.Fatal(err)
		}
	case "http":
		addr := fmt.Sprintf(":%d", *port)
		log.Printf("Starting HTTP server on %s", addr)
		if err := transport.RunHTTP(addr); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown transport: %s", *tr)
	}
}
