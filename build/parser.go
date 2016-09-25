package main

import (
	"strings"
)

type Parser struct {
	rawPrg string
	prg Program
	pc int
}

type Program struct {
	user string
	pw string
	cmds []Cmd
}

type Cmd interface {
}

type CmdReturn struct {
}

type CmdExit struct {
}

func newParser(p string) (*Parser) {
	return &Parser{rawPrg: p, pc: 0}
}

func (p *Parser) parse() int {
	lines := strings.Split(p.rawPrg, "\n")
	for i, l := range lines {
		words := splitLine(l)

		// check first line
		if i == 0 && words.areNext("as", "principal") {
			user := words.next()
			pw := words.next()
			if !words.parseError && isValidIdentifier(user) && isValidString(pw) {
				p.prg= Program{
					user: user,
					pw: pw,
					cmds: make([]Cmd, 1),
				}
			} else {
				return 2
			}
		}

		// check other lines
		switch (words.next()) {
		case "exit":
			p.prg.cmds = append(p.prg.cmds, CmdExit{})
		case "return":
			p.prg.cmds = append(p.prg.cmds, CmdReturn{})
		}

		// check if line parse was successful
		if words.parseError {
			return 2
		}
	}
	return 1
}

// >>>>>>>>>>> WORDS >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

type Words struct {
	items []string
	index int
	length int
	parseError bool
}

func splitLine(line string) (*Words) {
	i := strings.Fields(line)
	return &Words{items: i, index: -1, length: len(i)}
}

func (w *Words) next() string {
	if w.parseError { return "" }

	w.index = w.index + 1

	if w.index < w.length {
		return w.items[w.index]
	} else {
		w.parseError = true
		return ""
	}
}

func (w *Words) isNext(s string) bool {
	if w.parseError { return false }

	w.index = w.index + 1

	if w.index < w.length {
		return w.items[w.index] == s
	} else {
		w.parseError = true
		return false
	}
}

func (w *Words) areNext(ss... string) bool {
	if w.parseError { return false }

	result := true
	for _, s := range ss {
		w.index = w.index + 1

		if w.index >= w.length {
			w.parseError = true
			result = false
		} else if w.items[w.index] != s {
			result = false
		}
	}
	return result
}

func (w *Words) current() string {
	if w.parseError { return "" }

	if w.index < w.length {
		return w.items[w.index]
	} else {
		w.parseError = true
		return ""
	}
}

func (w *Words) isCurrent(s string) bool {
	if w.parseError { return false }

	if w.index < w.length {
		return w.items[w.index] == s
	} else {
		w.parseError = true
		return false
	}
}

func (w *Words) peek(i int) string {
	if w.parseError { return "" }

	if w.index + i < w.length {
		return w.items[w.index + i]
	} else {
		// do not throw parseError
		return ""
	}
}
