package main

type GlobalEnv struct {
	db *Database
}

type ProgramEnv struct {
	user string
	pw string
	globals *GlobalEnv
	results []Result
}

func NewGlobalEnv() *GlobalEnv {
	return &GlobalEnv{db: NewDatabase()}
}

func NewProgramEnv(ge *GlobalEnv) *ProgramEnv {
	return &ProgramEnv{globals: ge, results: make([]Result,0)}
}
