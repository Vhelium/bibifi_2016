package main

type GlobalEnv struct {
	db *Database
	dbSnapshot *Database
}

type ProgramEnv struct {
	principal string
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
