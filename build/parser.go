package main

import (
	"strings"
	"fmt"
)

type Parser struct {
	rawPrg string
	prg Program
	buf struct {
		tok Token // last read token
		lit string // last read literal
		n int      // buffer size (max=1)
	}
}

type Program struct {
	user string
	pw string
	cmds []Cmd
}

type Cmd interface {
	execute(*Environment) int
}

type CmdReturn struct {
}

type CmdExit struct {
}

func newParser(p string) (*Parser) {
	return &Parser{rawPrg: p}
}

// return codes:
// 0=success, 1=unfinished, 2=parseError
func ParseProgram(prg string) (int, *Program) {
	parser := newParser(prg)
	lines := strings.Split(prg, "\n")
	for i, l := range lines {
		fmt.Printf("Line(%d): ", i)
		if !parser.parseLine(i, l) {
			return 2, nil
		}
		fmt.Printf("\n")
	}
	return 0, &parser.prg
}

func (p *Parser) parseLine(i int, l string) bool {
	// get tokens
	tokenizer := NewTokenizer(strings.NewReader(l))

	// loop through tokens
	for {
		tok, lit := tokenizer.Scan()
		fmt.Printf("{%d, %s}, ", tok, lit)
		if tok == EOF {
			return true
		} else if tok == TERMINATE {
		}
	}

	return false
}
