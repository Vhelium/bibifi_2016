package main

// heavily based on: https://blog.gopheracademy.com/advent-2014/parsers-lexers/

import (
	"bufio"
	"io"
	"bytes"
	"strings"
)

type Token int

const (
	// Special tokens
	ILLEGAL Token = iota
	EOF

	// Literals
	IDENT
	STRING

	// Misc
	DOT				// .
	COMMA			// ,
	EMPTYLIST		// []
	EQUAL			// =
	ARROW			// ->
	BRACKET_OPEN		// {
	BRACKET_CLOSE	// }

	// Specific Keywords
	KV_TERMINATE
	KV_RETURN
	KV_EXIT

	KV_AS
	KV_PRINCIPAL
	KV_PASSWORD
	KV_DO
)

var eof = rune(0)

type Tokenizer struct {
	r *bufio.Reader
}

func NewTokenizer(r io.Reader) *Tokenizer {
	return &Tokenizer{r: bufio.NewReader(r)}
}

func (t *Tokenizer) read() rune {
	ch, _, err := t.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

func (t *Tokenizer) unread() { _ = t.r.UnreadRune() }

// returns next token and literal value
func (t *Tokenizer) Scan() (tok Token, lit string) {
	ch := t.read()

	// ignore whitespace
	if isWhitespace(ch) {
		for {
			ch = t.read()
			if ch == eof || !isWhitespace(ch) {
				break
			}
		}
	}
	if isLetter(ch) {
		t.unread()
		return t.scanIdent()
	}

	// otherwise read the individual char
	switch ch {
	case eof:
		return EOF, ""
	case '.':
		return DOT, "."
	case ',':
		return COMMA, ","
	case '{':
		return BRACKET_OPEN, "{"
	case '}':
		return BRACKET_CLOSE, "}"
	case '[':
		if t.read() == ']' {
			return EMPTYLIST, "[]"
		} else {
			return ILLEGAL, ""
		}
	case '=':
		return EQUAL, "="
	case '-':
		if t.read() == '>' {
			return ARROW, "->"
		} else {
			return ILLEGAL, ""
		}
	case '*':
		if t.read() == '*' && t.read() == '*' {
			return KV_TERMINATE, "***"
		} else {
			return ILLEGAL, ""
		}
	case '"':
		return t.scanString()
	}

	return ILLEGAL, string(ch)
}

func (t *Tokenizer) scanIdent() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(t.read())

	// Read every subsequent ident char into the buffer.
	// non-ident char and eof will cause loop to exit
	for {
		if ch := t.read(); ch == eof {
			break
		} else if !isLetter(ch) && !isDigit(ch) && ch != '_' {
			t.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}

	// check if it's a keyword here
	// and separate it from the regular idents:
	switch strings.ToUpper(buf.String()) {
	case "RETURN":
		return KV_RETURN, buf.String()
	case "EXIT":
		return KV_EXIT, buf.String()
	case "AS":
		return KV_AS, buf.String()
	case "PRINCIPAL":
		return KV_PRINCIPAL, buf.String()
	case "PASSWORD":
		return KV_PASSWORD, buf.String()
	case "DO":
		return KV_DO, buf.String()
	}

	if isValidIdentifier(buf.String()) {
		return IDENT, buf.String()
	} else {
		return ILLEGAL, "invalidIdent"
	}
}

func (t *Tokenizer) scanString() (tok Token, lit string) {
	var buf bytes.Buffer

	// read until end of string or eof
	for {
		if ch := t.read(); ch == eof {
			return ILLEGAL, ""
		} else if ch == '"' {
			if isValidString(buf.String()) {
				return STRING, buf.String()
			} else {
				return ILLEGAL, "invalidString"
			}
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	return ILLEGAL, ""
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch rune) bool {
	return (ch >= '0' && ch <= '9')
}
