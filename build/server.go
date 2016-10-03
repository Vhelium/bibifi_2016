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
var legitCommentRegex *regexp.Regexp
var globals *GlobalEnv

func main() {
	initialize()

	port := "6666"
	password := "admin"
	if len(os.Args) >= 2 {
		if isArgPortLegit(os.Args[1]) {
			port = os.Args[1]
		} else {
			log.Printf("Invalid port argument!")
			os.Exit(255)
		}
	}
	if len(os.Args) >= 3 {
		if isArgPwLegit(os.Args[2]) {
			password = os.Args[2]
		} else {
			log.Printf("Invalid pw argument")
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
		conn.SetReadDeadline(time.Now().Add(time.Minute * 3))
		conn.SetWriteDeadline(time.Now().Add(time.Minute * 5))

		tlen := 0;
		bufCmd := make ([]byte, 0, 4096)
		bufRcv := make ([]byte, 2048)
		O:
		for { // poll for input
			llen, err := conn.Read(bufRcv)
			tlen += llen
			if err != nil {
				if err != io.EOF {
					fmt.Println("Read error:", err)
				}
				conn.Write([]byte("{\"status\": \"TIMEOUT\"}"))
				conn.Close()
				break O
			} else {
				bufCmd = append(bufCmd, bufRcv[:llen]...)
			}
			if (tlen >= 3 && (string(bufCmd[tlen-3:tlen]) ==  "***")) ||
					(tlen >= 4 && (string(bufCmd[tlen-4:tlen]) ==  "***\n")) ||
					lineContainsTermination(string(bufCmd)) {
				r, s := executeProgram(string(bufCmd))
				results := fmt.Sprintf("%s\n", r)
				_, err := conn.Write([]byte(results))
				vcheck(err)
				conn.Close()
				if s == 0 {
					log.Printf("Shutting down server")
					os.Exit(0)
				}
				break
			}
		}
	}
}

func lineContainsTermination(p string) bool {
	lines := strings.Split(p, "\n")
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l) , "***") {
			return true
		}
	}
	return false
}

func executeProgram(p string) (string, int) {
	// parse
	res, prg := parseProgram(p)
	if res != 0 || prg == nil {
		return "{\"status\":\"FAILED\"}", -1
	}

	// backup db
	SnapshotDatabase(globals)

	// set up program env
	env := NewProgramEnv(globals)

	// execute
	res = prg.execute(env)
	if res != TERMINATED {
		// rollback db
		RollbackDatabase(globals)
	}

	result := ""
	for i, r := range env.results {
		res, e := json.Marshal(r)
		result += string(res)
		if i < len(env.results) - 1 {
			result += "\n"
		}
		if e != nil { fmt.Printf("err: ", e) }
	}

	return result, env.status_code
}

func initialize() {
	legitStringRegex = regexp.MustCompile(`[A-Za-z0-9_ ,;\.?!-]*`)
	legitIdentifierRegex = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_]*`)
	legitCommentRegex = regexp.MustCompile(`[A-Za-z0-9_ ,;\.?!-]*`)
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
	return len(s) < 65535 && s == legitStringRegex.FindString(s)
}

func isValidIdentifier(s string) bool {
	return len(s) <= 255 && s == legitIdentifierRegex.FindString(s)
}

func isValidComment(s string) bool {
	return s == legitCommentRegex.FindString(s)
}

func parseLine(l string) int {
	if strings.HasPrefix(strings.TrimLeft(l, " \t"), "***") {
		return 1
	} else {
		return 0
	}
}

func vcheck(err error) {
	if err != nil {
		fmt.Print(err)
	}
}
