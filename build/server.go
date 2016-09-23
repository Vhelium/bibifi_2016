package main

import (
	"fmt"
	"net"
	"os"
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
		vcheck(err)
		defer conn.Close()

		r := bufio.NewReader(conn)
		fmt.Printf(">>>>>>>>>>>> Program Start >>>>>>>>>>\n")
		for { // poll for input
			m, err := r.ReadString('\n')

			finished := parseLine(m)

			if err != nil {
				// e.g. EOF
				break
			}

			if finished {
				fmt.Printf(">>>>>>>>>>>> Program End >>>>>>>>>>>>\n")
				break
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

func parseLine(l string) bool {
	fmt.Printf("%s", l)
	return strings.HasPrefix(strings.TrimLeft(l, " \t"), "return") ||
			strings.HasPrefix(strings.TrimLeft(l, " \t"), "exit")
}

func vcheck(err error) {
	if err != nil {
		fmt.Print(err)
		os.Exit(2)
	}
}
