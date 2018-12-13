package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

func server() {
	addr, err := net.ResolveTCPAddr("tcp", ":7000")
	if err != nil {
		log.Fatalf("Error resolving tcp %v", err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatalf("[Error] Could not listen on port because: %v", err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[Error] Accepting a connection: %v", err)
			return
		}
		go func() {
			i := 0
			for range time.Tick(500 * time.Millisecond) {
				if _, err := conn.Write(append([]byte(fmt.Sprintf(`{"id":%d}`, i)), '\n')); err != nil {
					return
				}
			}
		}()
	}
}
func client(ctx context.Context) {
	cnn, err := net.DialTimeout("tcp", "127.0.0.1:7000", time.Second)
	if err != nil {
		log.Printf("[Error] Could not connect to server %v", err)
		return
	}

	start := time.Now()
	defer func() {
		err := cnn.Close() // I expect go routine with scan will be closed
		if err != nil {
			log.Printf("[ERROR] Closing connection  %v", err)
		}
		log.Printf("[DEBUG] Client finished  duration %s, connection closed ", time.Since(start))
	}()

	cnn.SetReadDeadline(time.Now().Add(time.Second))
	scanner := bufio.NewScanner(cnn)
	errs := make(chan error)
	lines := make(chan []byte)

	go func() {
		start := time.Now()
		defer func() {
			log.Printf("[DEBUG] !!!!! Read scan finished  duration %s ", time.Since(start)) // This never printed
		}()
		for scanner.Scan() {
			select {
			case lines <- scanner.Bytes():
				cnn.SetReadDeadline(time.Now().Add(time.Second))
			default:
			}
		}
		if err := scanner.Err(); err != nil {
			errs <- err
			return
		}
		errs <- io.EOF
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errs:
			if ctx.Err() == nil {
				log.Printf("[DEBUG] Read closed by err = %v", err)
			}
			return
		case <-lines:
			if ctx.Err() != nil {
				return
			}
		}
	}
}

func main() {
	go server()
	time.Sleep(time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	go client(ctx)
	time.Sleep(10 * time.Second)
	cancel()
	<-make(chan bool) //let's wait

}
