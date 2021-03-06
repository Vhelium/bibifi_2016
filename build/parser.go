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

type CmdComment struct {
}

type CmdReturn struct {
	expr Expr
}

type CmdExit struct {
}

type CmdAsPrincipal struct {
	principal string
	pw string
}

type CmdSet struct {
	ident string
	expr Expr
}

type CmdCreatePr struct {
	principal string
	pw string
}

type CmdChangePw struct {
	principal string
	pw string
}

type CmdLocal struct {
	ident string
	expr Expr
}

type CmdAppend struct {
	ident string
	expr Expr

}
type CmdForeach struct {
	identE string
	identL string
	expr Expr
}

type CmdSetDeleg struct {
	tgt string
	q string
	right AccessRight
	p string
}

type CmdDeleteDeleg struct {
	tgt string
	q string
	right AccessRight
	p string
}

type CmdDefaultDeleg struct {
	p string
}

type Expr interface {
	// []
	// {x=<val>, ..}
	// x
	// x.y
	// "string"
	eval(env *ProgramEnv) (int, *Value)
}

type ExprIdent struct {
	ident string
}
type ExprFieldAcc struct {
	ident string
	field string
}
type ExprString struct {
	val string
}
type ExprEmptyList struct {
}
type ExprRecord struct {
	fields map[string]Expr
}

func newParser(p string) (*Parser) {
	return &Parser{rawPrg: p}
}

func parseError(m string, p... interface{}) {
	fmt.Printf("[PERR]: " + m + "\n", p...)
}

// return codes:
// 0=success, 1=unfinished, 2=parseError
func parseProgram(prg string) (int, *Program) {
	parser := newParser(prg)
	lines := strings.Split(prg, "\n")
	for i, l := range lines {

		c, cmd := parser.parseLine(i, l)
		if c == 0 { // successful
			if cmd != nil {
				parser.prg.cmds = append(parser.prg.cmds, cmd)
			} else { // terminated
				return 0, &parser.prg
			}
		} else {
			return c, nil
		}
	}
	return 2, nil
}

// 0=success, 1=unfinished, 2=parseError
func (p *Parser) parseLine(i int, l string) (int, Cmd) {
	// get tokens
	tokenizer := NewTokenizer(strings.NewReader(l))

	// loop through tokens
	for {
		tok, _ := tokenizer.Scan()
		switch tok {
			case EOF: return 1, nil // not implemented function
			case KV_TERMINATE: return 0, nil
			case KV_EXIT: return p.parseCmdExit(tokenizer)
			case KV_RETURN:	return p.parseCmdReturn(tokenizer)
			case KV_AS: return p.parseCmdAsPrincipal(tokenizer)
			case KV_SET: return p.parseCmdSet(tokenizer)
			case KV_CREATE: return p.parseCmdCreatePr(tokenizer)
			case KV_CHANGE: return p.parseCmdChangePw(tokenizer)
			case KV_APPEND: return p.parseCmdAppend(tokenizer)
			case KV_LOCAL: return p.parseCmdLocal(tokenizer)
			case KV_FOREACH: return p.parseCmdForeach(tokenizer)
			case KV_DELETE: return p.parseCmdDeleteDeleg(tokenizer)
			case KV_DEFAULT: return p.parseCmdDefaultDeleg(tokenizer)
			case COMMENT: return p.parseCmdComment(tokenizer)
			default: return 1, nil
		}
	}

	return 2, nil
}

func (p *Parser) parseCmdExit(t *Tokenizer) (int, Cmd) {
	cmd := CmdExit{}
	return 0, cmd
}

func (p *Parser) parseCmdReturn(t *Tokenizer) (int, Cmd) {
	// get expression
	s, expr := p.parseExpr(t)
	if s == 0 {
		return 0, CmdReturn{expr: expr}
	}
	parseError("invalid CmdReturn")
	return 2, nil
}

func (p *Parser) parseCmdAsPrincipal(t *Tokenizer) (int, Cmd) {
	cmd := CmdAsPrincipal{}

	// read next keyword
	if tok, _ := t.Scan(); tok != KV_PRINCIPAL {
		parseError("expected PRINCIPAL in CmdAsPr")
		return 2, nil
	}

	// read principal
	tok, pr := t.Scan()
	if tok != IDENT {
		parseError("expected IDENT in CmdAsPr")
		return 2, nil
	}
	cmd.principal = pr

	// password token
	if tok, _ := t.Scan(); tok != KV_PASSWORD {
		parseError("expected PASSWORD in CmdAsPr")
		return 2, nil
	}

	// read pw
	tok, pw := t.Scan()
	if tok != STRING {
		parseError("expected STRING in CmdAsPr")
		return 2, nil
	}
	cmd.pw = pw

	// do token
	if tok, _ := t.Scan(); tok != KV_DO {
		parseError("expected DO in CmdAsPr")
		return 2, nil
	}

	return 0, cmd
}

func (p *Parser) parseCmdSet(t *Tokenizer) (int, Cmd) {
	cmd := CmdSet{}

	// get identifier
	tok, ident := t.Scan()
	if tok == KV_DELEGATION {
		return p.parseCmdSetDeleg(t)
	} else if tok != IDENT {
		parseError("expected IDENT in CmdSet")
		return 2, nil
	}
	cmd.ident = ident

	// read eq token
	if tok, _ := t.Scan(); tok != EQUAL {
		parseError("expected EQ in CmdSet")
		return 2, nil
	}

	// get expression
	s, expr := p.parseExpr(t)
	if s == 0 {
		return 0, CmdSet{ident: ident, expr: expr}
	}
	parseError("invalid CmdSet")
	return 2, nil
}

func (p *Parser) parseExpr(t *Tokenizer) (int, Expr) {
	tok, e := t.Scan()
	if tok == EMPTYLIST {
		return 0, ExprEmptyList{}
	} else if tok == BRACKET_OPEN {
		// parse record
		fields := make(map[string]Expr, 0)
		for {
			// read ident
			iTok, iExp := t.Scan()
			if iTok != IDENT {
				parseError("expected IDENT in record")
				return 2, nil
			}
			// read EQUAL
			if eTok, _ := t.Scan(); eTok != EQUAL {
				parseError("expected EQ in record")
				return 2, nil
			}
			// read <value>
			s, valExp := p.parseValue(t)
			if s == 0 {
				if _, exists := fields[iExp]; exists {
					parseError("duplicate key in record")
					return 2, nil
				}
				fields[iExp] = valExp
			} else {
				parseError("invalid value in record")
				return 2, nil
			}
			// check for comma, bracket or EOF/INVALID
			fTok, _ := t.Scan()
			if fTok == BRACKET_CLOSE {
				// successful parse
				return 0, ExprRecord{fields: fields}
			} else if fTok == COMMA {
				continue
			} else {
				parseError("invalid field in record")
				return 2, nil
			}
		}
	} else {
		t.Unscan(tok, e)
		return p.parseValue(t)
	}
}

func (p *Parser) parseValue(t *Tokenizer) (int, Expr) {
	tok, exp := t.Scan()
	if tok == STRING {
		return 0, ExprString{val: exp}
	} else if tok == IDENT {
		// check if its a field access
		if tok2, exp2 := t.Scan(); tok2 == DOT {
			if tok3, exp3 := t.Scan(); tok3 == IDENT {
				return 0, ExprFieldAcc{ident: exp, field: exp3}
			} else {
				parseError("Expected Identifier after '.'")
				return 2, nil
			}
		} else {
			t.Unscan(tok2, exp2)
			return 0, ExprIdent{ident: exp}
		}
	}
	parseError("invalid Value, got %s(%d) instead", exp, tok)
	return 2, nil
}

func(p *Parser) parseCmdCreatePr(t *Tokenizer) (int, Cmd) {
	cmd := CmdCreatePr{}

	// read PRINCIPAL token
	if tok, _ := t.Scan(); tok != KV_PRINCIPAL {
		parseError("expected PR in CmdCreatePr")
		return 2, nil
	}

	// get principal
	tok, pr := t.Scan()
	if tok != IDENT {
		parseError("expected IDENT in CmdCreatePr")
		return 2, nil
	}
	cmd.principal = pr

	// get pw
	tok, pw := t.Scan()
	if tok != STRING {
		parseError("expected STRING in CmdCreatePr")
		return 2, nil
	}
	cmd.pw = pw

	return 0, cmd
}

func(p *Parser) parseCmdChangePw(t *Tokenizer) (int, Cmd) {
	cmd := CmdChangePw{}

	// read PW token
	if tok, _ := t.Scan(); tok != KV_PASSWORD {
		parseError("expected PW in CmdChangePw")
		return 2, nil
	}

	// get principal
	tok, pr := t.Scan()
	if tok != IDENT {
		parseError("expected IDENT in CmdChangePw")
		return 2, nil
	}
	cmd.principal = pr

	// get pw
	tok, pw := t.Scan()
	if tok != STRING {
		parseError("expected STRING in CmdChangePw")
		return 2, nil
	}
	cmd.pw = pw

	return 0, cmd
}

func(p *Parser) parseCmdLocal(t *Tokenizer) (int, Cmd) {
	cmd := CmdLocal{}

	// get identifier
	tok, ident := t.Scan()
	if tok == KV_DELEGATION {
		return p.parseCmdSetDeleg(t)
	} else if tok != IDENT {
		parseError("expected IDENT in CmdLocal")
		return 2, nil
	}
	cmd.ident = ident

	// read eq token
	if tok, _ := t.Scan(); tok != EQUAL {
		parseError("expected EQ in CmdLocal")
		return 2, nil
	}

	// get expression
	s, expr := p.parseExpr(t)
	if s == 0 {
		return 0, CmdLocal{ident: ident, expr: expr}
	}
	parseError("invalid CmdLocal")
	return 2, nil
}

func(p *Parser) parseCmdAppend(t *Tokenizer) (int, Cmd) {
	// read to token
	if tok, _ := t.Scan(); tok != KV_TO {
		parseError("expected TO in CmdAppend")
		return 2, nil
	}

	// get identifier
	tok, ident := t.Scan()
	if tok != IDENT {
		parseError("expected IDENT in CmdAppend")
		return 2, nil
	}

	// read WITH token
	if tok, _ := t.Scan(); tok != KV_WITH{
		parseError("expected WITH in CmdAppend")
		return 2, nil
	}

	// get expression
	s, expr := p.parseExpr(t)
	if s == 0 {
		return 0, CmdAppend{ident: ident, expr: expr}
	}
	parseError("invalid CmdAppend")
	return 2, nil
}

func(p *Parser) parseCmdForeach(t *Tokenizer) (int, Cmd) {
	// get identifier
	tok, identE := t.Scan()
	if tok != IDENT {
		parseError("expected IDENT-E in CmdForeach")
		return 2, nil
	}

	// read IN token
	if tok, _ := t.Scan(); tok != KV_IN {
		parseError("expected WITH in CmdForeach")
		return 2, nil
	}

	// get identifier
	tok, identL := t.Scan()
	if tok != IDENT {
		parseError("expected IDENT-L in CmdForeach")
		return 2, nil
	}

	// read REPLACEWITH token
	if tok, _ := t.Scan(); tok != KV_REPLACEWITH {
		parseError("expected RPW in CmdForeach")
		return 2, nil
	}

	// get expression
	s, expr := p.parseExpr(t)
	if s == 0 {
		return 0, CmdForeach{identL: identL, identE: identE, expr: expr}
	}
	parseError("invalid CmdForeach")
	return 2, nil
}

func(*Parser) parseCmdSetDeleg(t *Tokenizer) (int, Cmd) {
	// get identifier
	tok, tgt := t.Scan()
	if tok != IDENT {
		parseError("expected IDENT-tgt in CmdSetDeleg")
		return 2, nil
	}

	// get identifier
	tok, q := t.Scan()
	if tok != IDENT {
		parseError("expected IDENT-q in CmdSetDeleg")
		return 2, nil
	}

	// get identifier
	tok, _ = t.Scan()
	var r AccessRight
	switch(tok) {
	case KV_READ: r = READ
	case KV_WRITE: r = WRITE
	case KV_DELEGATE: r = DELEGATE
	case KV_APPEND: r = APPEND
	default: parseError("expected IDENT-right in CmdSetDeleg"); return 2, nil
	}

	// read -> token
	if tok, _ := t.Scan(); tok != ARROW {
		parseError("expected ARROW in CmdSetDeleg")
		return 2, nil
	}

	// get identifier
	tok, p := t.Scan()
	if tok != IDENT {
		parseError("expected IDENT-L in CmdSetDeleg")
		return 2, nil
	}

	return 0, CmdSetDeleg{tgt, q, r, p}
}

func(*Parser) parseCmdDeleteDeleg(t *Tokenizer) (int, Cmd) {
	// read delegation token
	if tok, _ := t.Scan(); tok != KV_DELEGATION {
		parseError("expected ARROW in CmdDelDeleg")
		return 2, nil
	}

	// get identifier
	tok, tgt := t.Scan()
	if tok != IDENT {
		parseError("expected IDENT-tgt in CmdDelDeleg")
		return 2, nil
	}

	// get identifier
	tok, q := t.Scan()
	if tok != IDENT {
		parseError("expected IDENT-q in CmdDelDeleg")
		return 2, nil
	}

	// get identifier
	tok, _ = t.Scan()
	var r AccessRight
	switch(tok) {
	case KV_READ: r = READ
	case KV_WRITE: r = WRITE
	case KV_DELEGATE: r = DELEGATE
	case KV_APPEND: r = APPEND
	default: parseError("expected IDENT-right in CmdSetDeleg"); return 2, nil
	}

	// read -> token
	if tok, _ := t.Scan(); tok != ARROW {
		parseError("expected ARROW in CmdDelDeleg")
		return 2, nil
	}

	// get identifier
	tok, p := t.Scan()
	if tok != IDENT {
		parseError("expected IDENT-L in CmdDeleteDeleg")
		return 2, nil
	}

	return 0, CmdDeleteDeleg{tgt, q, r, p}
}

func(*Parser) parseCmdDefaultDeleg(t *Tokenizer) (int, Cmd) {
	// read deleg token
	if tok, _ := t.Scan(); tok != KV_DELEGATOR {
		parseError("expected deleg in CmdDefDeleg")
		return 2, nil
	}

	// read EQ token
	if tok, _ := t.Scan(); tok != EQUAL {
		parseError("expected deleg in CmdDefDeleg")
		return 2, nil
	}

	// get identifier
	tok, p := t.Scan()
	if tok != IDENT {
		parseError("expected IDENT in CmdDefDeleg")
		return 2, nil
	}
	return 0, CmdDefaultDeleg{p}
}

func(*Parser) parseCmdComment(t *Tokenizer) (int, Cmd) {
	return 0, CmdComment{}
}
