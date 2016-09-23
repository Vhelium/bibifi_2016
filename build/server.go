package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"log"
	"bufio"
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
	conn, err := ln.Accept()
	vcheck(err)
	defer conn.Close()

	r := bufio.NewReader(conn)
	for {
		m, err := r.ReadString('\n')

		finished := parseLine(m)

		if err != nil {
			// e.g. EOF
			break
		}

		if finished {
			break
		}

	}
	fmt.Printf(">>>>>>>>>> END OF PROGRAM >>>>>>>>")
}

func parseLine(l string) (eof bool) {
	fmt.Printf("r:\t%s", l)
	return strings.HasPrefix(strings.TrimLeft(l, " \t"), "return")
}

func vcheck(err error) {
	if err != nil {
		fmt.Print(err)
		os.Exit(2)
	}
}
