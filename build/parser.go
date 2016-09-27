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
	cmds []Cmd
}

type Cmd interface {
	execute(*ProgramEnv) int
}

type CmdReturn struct {
}

type CmdExit struct {
}

type CmdAsPrincipal struct {
	user string
	pw string
}

func newParser(p string) (*Parser) {
	return &Parser{rawPrg: p}
}

// return codes:
// 0=success, 1=unfinished, 2=parseError
func parseProgram(prg string) (int, *Program) {
	parser := newParser(prg)
	lines := strings.Split(prg, "\n")
	for i, l := range lines {
		fmt.Printf("Line(%d): ", i)

		c, cmd := parser.parseLine(i, l)
		if c == 0 { // successful
			if cmd != nil {
				parser.prg.cmds = append(parser.prg.cmds, cmd)
			} else { // terminated
				fmt.Printf("\n")
				return 0, &parser.prg
			}
		} else {
			fmt.Printf("\n")
			return c, nil
		}
		fmt.Printf("\n")
	}
	return 2, nil
}

// 0=success, 1=unfinished, 2=parseError
func (p *Parser) parseLine(i int, l string) (int, Cmd) {
	// get tokens
	tokenizer := NewTokenizer(strings.NewReader(l))

	// loop through tokens
	for {
		tok, lit := tokenizer.Scan()
		fmt.Printf("{%d, %s}, ", tok, lit)
		if tok == EOF {
			return 1, nil // not implemented function
		} else if tok == KV_TERMINATE {
			return 0, nil
		} else if tok == KV_EXIT {
			return p.parseCmdExit(tokenizer)
		} else if tok == KV_RETURN {
			return p.parseCmdReturn(tokenizer)
		} else if tok == KV_AS {
			return p.parseCmdAsPrincipal(tokenizer)
		}
	}

	return 2, nil
}

func (p *Parser) parseCmdExit(t *Tokenizer) (int, Cmd) {
	cmd := CmdExit{}
	return 0, cmd
}

func (p *Parser) parseCmdReturn(t *Tokenizer) (int, Cmd) {
	cmd := CmdReturn{}
	return 0, cmd
}

func (p *Parser) parseCmdAsPrincipal(t *Tokenizer) (int, Cmd) {
	cmd := CmdAsPrincipal{}

	// read next keyword
	if tok, _ := t.Scan(); tok != KV_PRINCIPAL {
		return 2, nil
	}

	// read user
	tok, user := t.Scan()
	if tok != IDENT { return 2, nil }
	cmd.user = user

	// password token
	if tok, _ := t.Scan(); tok != KV_PASSWORD {	return 2, nil}

	// read user
	tok, pw := t.Scan()
	if tok != STRING { return 2, nil }
	cmd.pw = pw

	// do token
	if tok, _ := t.Scan(); tok != KV_DO { return 2, nil }

	return 0, cmd
}
