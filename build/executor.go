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
	Output interface{}	`json:"output,omitempty"`
}

type Value struct {
	s int
	val string
	vals map[string]string
}

func (val ExprString) eval(env *ProgramEnv) (int, *Value) {
	return DB_VAR_FOUND, &Value{s:0, val: val.val}
}

func (val ExprFieldAcc) eval(env *ProgramEnv) (int, *Value) {
	s, v := env.globals.db.getFieldValueFor(val.ident, val.field, env.user)
	if s == DB_VAR_FOUND {
		return DB_VAR_FOUND, &Value{s:0, val: v}
	}
	return s, nil
}

func (val ExprIdent) eval(env *ProgramEnv) (int, *Value) {
	s, v := env.globals.db.getVarValueFor(val.ident, env.user)
	if s == DB_VAR_FOUND {
		return DB_VAR_FOUND, &Value{s:0, val: v}
	}
	return s, nil
}

func (expr ExprEmptyList) eval(env *ProgramEnv) (int, *Value) {
	return DB_VAR_FOUND, &Value{s: 1, vals: make(map[string]string, 0)}
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
	return DB_VAR_FOUND, &Value{s: 1, vals: f}
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
	env.results = append(env.results, Result{Status: "RETURNING"})
	return TERMINATED
}

func (cmd CmdAsPrincipal) execute(env *ProgramEnv) int {
	env.user = cmd.user
	env.pw = cmd.pw
	if env.globals.db.isLoginCorrect(env.user, env.pw) {
		return SUCCESS
	} else {
		env.results = append(env.results, Result{Status: "DENIED"})
		return DENIED
	}
}

func (cmd CmdSet) execute(env *ProgramEnv) int {
	s, val := cmd.expr.eval(env)
	if s == DB_VAR_FOUND {
		// check if variable exists && user has WRITE rights on it
		if env.globals.db.doesGlobalVarExist(cmd.ident) {
			if env.globals.db.hasUserPrivilege(cmd.ident, env.user, WRITE) {
				env.globals.db.setGlobalVar(cmd.ident, val)
			}
		} else {
			// otherwise, create new w/ corresponding rights
			env.globals.db.setGlobalVar(cmd.ident, val)
			env.globals.db.setAllDelegations(cmd.ident, "admin", env.user)
		}

		env.results = append(env.results, Result{Status: "SET"})
		return SUCCESS
	} else if s == DB_INSUFFICIENT_RIGHTS {
		env.results = append(env.results, Result{Status: "DENIED"})
		return DENIED
	} else {
		env.results = append(env.results, Result{Status: "FAILED"})
		return FAILED
	}
}

// to fail, just assign: env.results := []Result{ {"status":"DENIED"} }
