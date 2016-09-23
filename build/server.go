package main

import (
	"fmt"
	"net"
	"os"
	"io"
	"strconv"
	"log"
	"strings"
)

func main() {
	port := "6666"
	password := "admin"
	if len(os.Args) >= 2 {
		p, err := strconv.Atoi(os.Args[1])
		if err == nil && p >= 1024 && p <= 65535 {
			port = os.Args[1]
		} else {
			log.Fatal("Invalid port argument")
			os.Exit(255)
		}
	}
	if len(os.Args) >= 3 {
		password = os.Args[2]
	}

	log.Printf("Starting server on port %s w/ password %s", port, password)

	ln, err := net.Listen("tcp", ":"+port)
	vcheck(err)

	for {
		conn, err := ln.Accept()
		vcheck(err)
		defer conn.Close()

		bufCmd := make ([]byte, 0, 4096)
		bufRcv := make ([]byte, 2048)
		for {
			len, err := conn.Read(bufRcv)
			if err != nil {
				if err != io.EOF {
					fmt.Println("Read error:", err)
				}
				break
			}
			bufCmd = append(bufCmd, bufRcv[:len]...)
			if isProgramComplete(&bufCmd) {
				break
			}
		}
		fmt.Printf("msg:\n%s", string(bufCmd))
	}
}

func isProgramComplete(buf *[]byte) (bool) {
	lines := strings.Split(string(*buf), "\n")
	for _,l := range lines {
		if isLineTerminating(l) {
			return true
		}
	}
	return false
}

func isLineTerminating(l string) (bool) {
	return strings.HasPrefix(strings.TrimLeft(l, " \t"), "return") ||
			strings.HasPrefix(strings.TrimLeft(l, " \t"), "exit")
}

func vcheck(err error) {
	if err != nil {
		fmt.Print(err)
		os.Exit(2)
	}
}
