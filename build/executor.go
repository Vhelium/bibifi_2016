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

// to fail, just assign: env.results := []Result{ {"status":"DENIED"} }
