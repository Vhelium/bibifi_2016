package main

import (
	"fmt"
	"net"
	"os"
	"time"
	"io"
	"strconv"
	"log"
	"strings"
	"regexp"
	"encoding/json"
)

var legitStringRegex *regexp.Regexp
var legitIdentifierRegex *regexp.Regexp
var globals *GlobalEnv

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
	// add admin principal to db
	globals.db.addUser("admin", password)

	ln, err := net.Listen("tcp", ":"+port)
	vcheck(err)

	for { // poll for requests
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Client aborted: \n", err)
		}
		// set timeouts
		conn.SetReadDeadline(time.Now().Add(time.Second * 10))
		conn.SetWriteDeadline(time.Now().Add(time.Second * 10))

		tlen := 0;
		bufCmd := make ([]byte, 0, 4096)
		bufRcv := make ([]byte, 2048)
		fmt.Printf(">>>>>>>>>>>> Program Start >>>>>>>>>>\n")
		for { // poll for input
			llen, err := conn.Read(bufRcv)
			tlen += llen
			if err != nil {
				if err != io.EOF {
					fmt.Println("Read error:", err)
				}
				break
			}

			// TODO: make it more efficient (i.e. direct copy)
			bufCmd = append(bufCmd, bufRcv[:llen]...)

			if tlen >= 3 && (string(bufCmd[tlen-3:tlen]) ==  "***" ||
					string(bufCmd[tlen-4:tlen]) ==  "***\n") {
				fmt.Printf(string(bufCmd))
				fmt.Printf(">>>>>>>>>>>> Program End >>>>>>>>>>>>\n")

				r := executeProgram(string(bufCmd))
				results := fmt.Sprintf("%s\n", r)
				conn.Write([]byte(results))
				conn.Close()
				break
			}
		}
	}
}

func executeProgram(p string) string {
	// parse
	res, prg := parseProgram(p)
	if res != 0 || prg == nil {
		return "{\"status\":\"FAILED\"}"
	}

	// backup db
	SnapshotDatabase(globals)

	// env
	env := NewProgramEnv(globals)

	// execute
	res = prg.execute(env)
	if res != TERMINATED {
		// rollback db
		RollbackDatabase(globals)
	}

	result := ""
	for _, r := range env.results {
		res, e := json.Marshal(r)
		result += string(res) + "\n"
		if e != nil { fmt.Printf("err: ", e) }
		fmt.Printf("ress: %s\n", res)
	}

	env.globals.db.printDB()

	return result
}

func initialize() {
	legitStringRegex = regexp.MustCompile(`[A-Za-z0-9_ ,;\.?!-]*`)
	legitIdentifierRegex = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_]*`)
	globals = NewGlobalEnv()
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
	return len(pw) <= 4096 && isValidString(pw)
}

func isValidString(s string) bool {
	//return len(s) <= 65535 && ...
	return s == legitStringRegex.FindString(s);
}

func isValidIdentifier(s string) bool {
	return len(s) <= 255 && s == legitStringRegex.FindString(s);
}

func parseLine(l string) int {
	fmt.Printf("%s", l)
	if strings.HasPrefix(strings.TrimLeft(l, " \t"), "***") {
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
