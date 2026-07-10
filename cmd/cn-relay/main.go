// cn-relay listens on a loopback port and forwards every connection to a
// target host on the same port. It exists because the Claude Notifier VS
// Code extension always pushes remoteAudio events to 127.0.0.1:<port> from
// inside whatever machine or container Claude Code runs in — that target is
// hardcoded (its settings sync only exposes "enabled" and "port", never a
// host), so nothing can redirect it to an external host by configuration
// alone. Running this inside the same container as the dev environment,
// rather than a sidecar joined via network_mode, satisfies that loopback
// requirement without adding a second container per project.
package main

import (
	"flag"
	"io"
	"log"
	"net"
)

func main() {
	port := flag.String("port", "47291", "loopback port to listen on")
	target := flag.String("target", "host.docker.internal", "host to forward connections to")
	flag.Parse()

	addr := "127.0.0.1:" + *port
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("cn-relay: listen %s: %v", addr, err)
	}
	log.Printf("cn-relay: listening on %s -> %s:%s", addr, *target, *port)

	for {
		client, err := ln.Accept()
		if err != nil {
			log.Printf("cn-relay: accept: %v", err)
			continue
		}
		go relay(client, *target, *port)
	}
}

func relay(client net.Conn, targetHost, port string) {
	defer client.Close()

	upstream, err := net.Dial("tcp", net.JoinHostPort(targetHost, port))
	if err != nil {
		log.Printf("cn-relay: dial %s:%s: %v", targetHost, port, err)
		return
	}
	defer upstream.Close()

	done := make(chan struct{}, 2)
	go func() { io.Copy(upstream, client); done <- struct{}{} }()
	go func() { io.Copy(client, upstream); done <- struct{}{} }()
	<-done
}
