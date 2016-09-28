package main

import (
)

const (
	SUCCESS = 0
	FAILED = 1
	DENIED = 2
	TERMINATED = 3
)

type Result struct {
	Status string		`json:"status"`
	Output string		`json:"output,omitempty"`
}

type Value struct {
	mode int
	val string
	vals map[string]string
	list []*Value
}

func (val ExprString) eval(env *ProgramEnv) (int, *Value) {
	return DB_VAR_FOUND, &Value{mode:0, val: val.val}
}

func (val ExprFieldAcc) eval(env *ProgramEnv) (int, *Value) {
	s, v := env.globals.db.getFieldValueFor(val.ident, val.field, env.user)
	if s == DB_VAR_FOUND {
		return DB_VAR_FOUND, &Value{mode:0, val: v}
	}
	return s, nil
}

func (val ExprIdent) eval(env *ProgramEnv) (int, *Value) {
	s, v := env.globals.db.getVarValueFor(val.ident, env.user)
	if s == DB_VAR_FOUND {
		return DB_VAR_FOUND, &Value{mode:0, val: v}
	}
	return s, nil
}

func (expr ExprEmptyList) eval(env *ProgramEnv) (int, *Value) {
	return DB_VAR_FOUND, &Value{mode: VAR_MODE_LIST, list: make([]*Value, 0)}
}

func (expr ExprRecord) eval(env *ProgramEnv) (int, *Value) {
	f := make(map[string]string,0)
	for k, vals := range expr.fields {
		s, v := vals.eval(env)
		if s != DB_VAR_FOUND {
			return s, nil
		} else {
			// must evaluate to string
			f[k] = v.val
		}
	}
	return DB_VAR_FOUND, &Value{mode: VAR_MODE_RECORD, vals: f}
}

func (p Program) execute(env *ProgramEnv) int {
	for _,cmd := range p.cmds {
		r := cmd.execute(env)
		if r != SUCCESS {
			return r
		}
	}
	return FAILED // didn't terminate..
}

func (cmd CmdExit) execute(env *ProgramEnv) int {
	env.results = append(env.results, Result{Status: "EXITING"})
	return TERMINATED
}

func (cmd CmdReturn) execute(env *ProgramEnv) int {
	_, o := cmd.expr.eval(env)
	env.results = append(env.results, Result{
		Status: "RETURNING",
		Output: printValue(NewEntryVar("", o)) , //TODO: right format
	})
	return TERMINATED
}

func (cmd CmdAsPrincipal) execute(env *ProgramEnv) int {
	env.user = cmd.user
	env.pw = cmd.pw
	if env.globals.db.isLoginCorrect(env.user, env.pw) {
		return SUCCESS
	} else {
		env.results = []Result{ Result{Status: "DENIED"} }
		return DENIED
	}
}

func (cmd CmdSet) execute(env *ProgramEnv) int {
	s, val := cmd.expr.eval(env)
	if s == DB_VAR_FOUND {
		set := env.globals.db.setGlobalVarFor(cmd.ident, val, env.user)
		if set == DB_SUCCESS {
			env.results = append(env.results, Result{Status: "SET"})
			return SUCCESS
		} else {
			env.results = []Result{ Result{Status: "DENIED"} }
			return DENIED
		}
	} else if s == DB_INSUFFICIENT_RIGHTS {
		env.results = []Result{ Result{Status: "DENIED"} }
		return DENIED
	} else {
		env.results = []Result{ Result{Status: "FAILED"} }
		return FAILED
	}
}

func (cmd CmdCreatePr) execute(env *ProgramEnv) int { /* TODO */ return SUCCESS }
func (cmd CmdChangePw) execute(env *ProgramEnv) int { /* TODO */ return SUCCESS }
func (cmd CmdAppend) execute(env *ProgramEnv) int { /* TODO */ return SUCCESS }
func (cmd CmdLocal) execute(env *ProgramEnv) int { /* TODO */ return SUCCESS }
func (cmd CmdForeach) execute(env *ProgramEnv) int { /* TODO */ return SUCCESS }
func (cmd CmdSetDeleg) execute(env *ProgramEnv) int { /* TODO */ return SUCCESS }
func (cmd CmdDeleteDeleg) execute(env *ProgramEnv) int { /* TODO */ return SUCCESS }
func (cmd CmdDefaultDeleg) execute(env *ProgramEnv) int { /* TODO */ return SUCCESS }

// to fail, just assign: env.results := []Result{ Result{"status":"DENIED"} }
