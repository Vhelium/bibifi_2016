package main

import (
	"fmt"
	"net"
	"os"
	"time"
	"bufio"
	"strconv"
	"log"
	"strings"
	"regexp"
)

var legitStringRegex *regexp.Regexp;

func main() {
	initialize()

	port := "6666"
	password := "admin"
	if len(os.Args) >= 2 {
		if isArgPortLegit(os.Args[1]) {
			port = os.Args[1]
		} else {
			log.Fatal("Invalid port argument")
			os.Exit(255)
		}
	}
	if len(os.Args) >= 3 {
		if isArgPwLegit(os.Args[2]) {
			password = os.Args[2]
		} else {
			log.Fatal("Invalid pw argument")
			os.Exit(255)
		}
	}

	log.Printf("Starting server on port %s w/ password %s", port, password)

	ln, err := net.Listen("tcp", ":"+port)
	vcheck(err)

	for { // poll for requests
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Client aborted: \n", err)
		}
		// set timeouts
		conn.SetReadDeadline(time.Now().Add(time.Second * 5))
		conn.SetWriteDeadline(time.Now().Add(time.Second * 5))

		r := bufio.NewReader(conn)
		ch := make(chan string, 1) // buffered line
		timeout := make(chan bool, 1) // timeout

		fmt.Printf(">>>>>>>>>>>> Program Start >>>>>>>>>>\n")
		P:
		for { // poll for input
			go func() {
				m, err := r.ReadString('\n')
				if err == nil {
					ch <- m
				}
			}()
			go func() {
				time.Sleep(5 * time.Second)
				timeout <- true
			}()

			select {
			case m:= <-ch:
				// we got some data
				status := parseLine(m)

				if status != 0 {
					fmt.Printf(">>>>>>>>>>>> Program End >>>>>>>>>>>>\n")
					conn.Write([]byte("okthxbye\n"))
					conn.Close()
					break
				}
			case <-timeout:
				// read timed out
				log.Printf("Client timed out.")
				break P
			}
		}
	}
}

func initialize() {
	legitStringRegex = regexp.MustCompile(`[A-Za-z0-9_ ,;\.?!-]*`)
}

func isArgPortLegit(port string) bool {
	// check for '0' prefix and len <= 4096
	if port[0] == '0' || len(port) > 4096 {
		return false
	}
	// check if legit decimal
	p, err := strconv.Atoi(os.Args[1])
	if err == nil && p >= 1024 && p <= 65535 {
		return true
	}
	return false
}

func isArgPwLegit(pw string) bool {
	return len(pw) <= 4096 && isStringLegit(pw)
}

func isStringLegit(s string) bool {
	//return len(s) <= 65535 && ...
	return s == legitStringRegex.FindString(s);
}

func parseLine(l string) int {
	fmt.Printf("%s", l)
	if strings.HasPrefix(strings.TrimLeft(l, " \t"), "return") ||
			strings.HasPrefix(strings.TrimLeft(l, " \t"), "exit") {
		return 1
	} else {
		return 0
	}
}

func vcheck(err error) {
	if err != nil {
		fmt.Print(err)
		os.Exit(2)
	}
}
