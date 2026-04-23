// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package portforward

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// ProxyPair defines a port forwarding pair
type ProxyPair struct {
	ListenAddr string
	TargetAddr string
}

// DefaultPairs returns the standard MageBox port forwarding pairs
func DefaultPairs() []ProxyPair {
	return []ProxyPair{
		{ListenAddr: "127.0.0.1:80", TargetAddr: "127.0.0.1:8080"},
		{ListenAddr: "127.0.0.1:443", TargetAddr: "127.0.0.1:8443"},
		{ListenAddr: "[::1]:80", TargetAddr: "[::1]:8080"},
		{ListenAddr: "[::1]:443", TargetAddr: "[::1]:8443"},
	}
}

// RunProxy starts a TCP proxy that forwards traffic between the given pairs.
// It blocks until a SIGTERM/SIGINT is received. Designed to be run as a
// LaunchDaemon — launchd keeps it alive, so reboots and sleep/wake just work.
func RunProxy(pairs []ProxyPair) error {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	var wg sync.WaitGroup
	listeners := make([]net.Listener, 0, len(pairs))

	for _, p := range pairs {
		ln, err := net.Listen("tcp", p.ListenAddr)
		if err != nil {
			// Close any listeners we already opened
			for _, l := range listeners {
				l.Close()
			}
			return fmt.Errorf("listen %s: %w", p.ListenAddr, err)
		}
		listeners = append(listeners, ln)
		logger.Printf("forwarding %s -> %s", p.ListenAddr, p.TargetAddr)

		target := p.TargetAddr
		wg.Add(1)
		go func(ln net.Listener) {
			defer wg.Done()
			for {
				conn, err := ln.Accept()
				if err != nil {
					return // listener closed
				}
				go forward(conn, target, logger)
			}
		}(ln)
	}

	// Wait for shutdown signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	s := <-sig
	logger.Printf("received %s, shutting down", s)

	// Close all listeners (causes Accept to return error, goroutines exit)
	for _, ln := range listeners {
		ln.Close()
	}
	wg.Wait()

	return nil
}

// forward proxies data between a client connection and the target
func forward(client net.Conn, targetAddr string, logger *log.Logger) {
	defer client.Close()

	target, err := net.Dial("tcp", targetAddr)
	if err != nil {
		// Target not reachable (nginx not running) — close silently
		return
	}
	defer target.Close()

	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(target, client)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(client, target)
		done <- struct{}{}
	}()

	// Wait for either direction to finish
	<-done
}
