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
	mode int
	val string
	vals map[string]string
	list []*Value
}

func formatOutput(v *Value) interface{} {
	if v.mode == 0 {
		return v.val
	} else if v.mode == VAR_MODE_RECORD {
		return v.vals
	} else {
		list := make([]interface{},0)
		for _,l := range v.list {
			item := formatOutput(l)
			list = append(list, item)
		}
		return list
	}}

func (val ExprString) eval(env *ProgramEnv) (int, *Value) {
	return DB_VAR_FOUND, &Value{mode:0, val: val.val}
}

func (val ExprFieldAcc) eval(env *ProgramEnv) (int, *Value) {
	s, v := env.getFieldValueForWith(val.ident, val.field, env.principal, READ)
	if s == DB_VAR_FOUND {
		return DB_VAR_FOUND, &Value{mode:0, val: v}
	}
	return s, nil
}

func (val ExprIdent) eval(env *ProgramEnv) (int, *Value) {
	s, v := env.getVarValueForWith(val.ident, env.principal, READ)
	if s == DB_VAR_FOUND {
		return DB_VAR_FOUND, v
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
		Output: formatOutput(o),
	})
	return TERMINATED
}

func (cmd CmdAsPrincipal) execute(env *ProgramEnv) int {
	env.principal = cmd.principal
	env.pw = cmd.pw
	if env.globals.db.isLoginCorrect(env.principal, env.pw) {
		return SUCCESS
	} else {
		env.results = []Result{ Result{Status: "DENIED"} }
		return DENIED
	}
}

func (cmd CmdSet) execute(env *ProgramEnv) int {
	s, val := cmd.expr.eval(env)
	if s == DB_VAR_FOUND {
		set := env.setVarForWith(cmd.ident, val, env.principal, WRITE)
		if set == DB_SUCCESS {
			env.results = append(env.results, Result{Status: "SET"})
			return SUCCESS
		} else if s == DB_INSUFFICIENT_RIGHTS {
			env.results = []Result{ Result{Status: "DENIED"} }
			return DENIED
		} else {
			env.results = []Result{ Result{Status: "FAILED"} }
			return FAILED
		}
	} else if s == DB_INSUFFICIENT_RIGHTS {
		env.results = []Result{ Result{Status: "DENIED"} }
		return DENIED
	} else {
		env.results = []Result{ Result{Status: "FAILED"} }
		return FAILED
	}
}

func (cmd CmdCreatePr) execute(env *ProgramEnv) int {
	if env.doesUserExist(cmd.principal) {
		env.results = []Result{ Result{Status: "FAILED"} }
		return FAILED
	}
	if !env.globals.db.isUserAdmin(env.principal) {
		env.results = []Result{ Result{Status: "DENIED"} }
		return DENIED
	}
	env.addUser(cmd.principal, cmd.pw)
	env.results = append(env.results, Result{Status: "CREATE_PRINCIPAL"})
	return SUCCESS
}

func (cmd CmdChangePw) execute(env *ProgramEnv) int {
	if !env.doesUserExist(cmd.principal) {
		env.results = []Result{ Result{Status: "FAILED"} }
		return FAILED
	}
	if env.globals.db.isUserAdmin(env.principal) || env.principal == cmd.principal {
		env.globals.db.changePassword(cmd.principal, cmd.pw)
		env.results = append(env.results, Result{Status: "CHANGE_PASSWORD"})
		return SUCCESS
	} else {
		env.results = []Result{ Result{Status: "DENIED"} }
		return DENIED
	}
}

func (cmd CmdLocal) execute(env *ProgramEnv) int {
	s, val := cmd.expr.eval(env)
	if s == DB_VAR_FOUND {
		set := env.setLocalVar(cmd.ident, val)
		if set == DB_SUCCESS {
			env.results = append(env.results, Result{Status: "LOCAL"})
			return SUCCESS
		} else if s == DB_INSUFFICIENT_RIGHTS {
			env.results = []Result{ Result{Status: "DENIED"} }
			return DENIED
		} else {
			env.results = []Result{ Result{Status: "FAILED"} }
			return FAILED
		}
	} else if s == DB_INSUFFICIENT_RIGHTS {
		env.results = []Result{ Result{Status: "DENIED"} }
		return DENIED
	} else {
		env.results = []Result{ Result{Status: "FAILED"} }
		return FAILED
	}
}

func (cmd CmdAppend) execute(env *ProgramEnv) int {
	s, exprVal := cmd.expr.eval(env)
	if s == DB_VAR_FOUND {
		sx, x := env.getVarValueForWith(cmd.ident, env.principal, APPEND, WRITE)
		if sx == DB_VAR_FOUND && x.mode == VAR_MODE_LIST {
			// append expr to cmd.ident
			switch exprVal.mode {
				case VAR_MODE_SINGLE: fallthrough
				case VAR_MODE_RECORD: // append
					env.appendVarToListFor(cmd.ident, exprVal, env.principal)
				case VAR_MODE_LIST: // concat
					env.concatListToListFor(cmd.ident, exprVal, env.principal)
			}
			env.results = append(env.results, Result{Status: "APPEND"})
			return SUCCESS
		} else if sx == DB_INSUFFICIENT_RIGHTS {
			env.results = []Result{ Result{Status: "DENIED"} }
			return DENIED
		} else {
			env.results = []Result{ Result{Status: "FAILED"} }
			return FAILED
		}
	} else if s == DB_INSUFFICIENT_RIGHTS {
		env.results = []Result{ Result{Status: "DENIED"} }
		return DENIED
	} else {
		env.results = []Result{ Result{Status: "FAILED"} }
		return FAILED
	}
}

func (cmd CmdForeach) execute(env *ProgramEnv) int {
	// check if local var already exists
	if env.doesVarExist(cmd.identE) {
		env.results = []Result{ Result{Status: "FAILED"} }
		return FAILED
	}
	// get list with r/w permission
	sl, l := env.getVarValueForWith(cmd.identL, env.principal, READ, WRITE)
	if sl == DB_VAR_FOUND && l.mode == VAR_MODE_LIST {
		// create new list var
		newList := &Value{mode: VAR_MODE_LIST, list: make([]*Value, 0)}
		// loop all list entries
		for _, item := range l.list {
			// create tmp local variable (we know it won't fail)
			env.setLocalVar(cmd.identE, item)
			// evaluate expr
			si, expr := cmd.expr.eval(env)
			if si == DB_VAR_FOUND {
				// apply expr on local var && append to list
				newList.list = append(newList.list, expr)
			} else if si == DB_INSUFFICIENT_RIGHTS {
				env.results = []Result{ Result{Status: "DENIED"} }
				return DENIED
			} else {
				env.results = []Result{ Result{Status: "FAILED"} }
				return FAILED
			}
			// discard tmp local variable
			env.discardLocalVar(cmd.identE)
		}
		// write new list in old location
		env.setVarForWith(cmd.identL, newList, env.principal) // must succeed
		env.results = append(env.results, Result{Status: "FOREACH"})
		return SUCCESS
	} else if sl == DB_INSUFFICIENT_RIGHTS {
		env.results = []Result{ Result{Status: "DENIED"} }
		return DENIED
	} else {
		env.results = []Result{ Result{Status: "FAILED"} }
		return FAILED
	}
}

func (cmd CmdSetDeleg) execute(env *ProgramEnv) int {
	if !env.doesUserExist(cmd.q) || !env.doesUserExist(cmd.q) {
		env.results = []Result{ Result{Status: "FAILED"} }
		return FAILED
	}
	s := env.setDelegation(cmd.tgt, cmd.q, cmd.p, cmd.right)
	switch s {
	case DB_SUCCESS:
		env.results = append(env.results, Result{Status: "SET_DELEGATION"})
		return SUCCESS
	case DB_INSUFFICIENT_RIGHTS:
		env.results = []Result{ Result{Status: "DENIED"} }
		return DENIED
	default:
		env.results = []Result{ Result{Status: "FAILED"} }
		return FAILED
	}
}
func (cmd CmdDeleteDeleg) execute(env *ProgramEnv) int {
	if !env.doesUserExist(cmd.q) || !env.doesUserExist(cmd.q) {
		env.results = []Result{ Result{Status: "FAILED"} }
		return FAILED
	}
	s := env.deleteDelegation(cmd.tgt, cmd.q, cmd.p, cmd.right)
	switch s {
	case DB_SUCCESS:
		env.results = append(env.results, Result{Status: "SET_DELEGATION"})
		return SUCCESS
	case DB_INSUFFICIENT_RIGHTS:
		env.results = []Result{ Result{Status: "DENIED"} }
		return DENIED
	default:
		env.results = []Result{ Result{Status: "FAILED"} }
		return FAILED
	}
}

func (cmd CmdDefaultDeleg) execute(env *ProgramEnv) int {
	if !env.doesUserExist(cmd.p) {
		env.results = []Result{ Result{Status: "FAILED"} }
		return FAILED
	} else if !env.globals.db.isUserAdmin(env.principal) {
		env.results = []Result{ Result{Status: "DENIED"} }
		return DENIED
	}
	env.setDefaultDelegator(cmd.p)
	return SUCCESS
}

// to fail, just assign: env.results := []Result{ Result{"status":"DENIED"} }
