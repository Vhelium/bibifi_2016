package main

type Database struct {
	users map[string]*EntryUser					// 1:1
	delegations map[string][]*EntryDelegation	// 1:N
	vars map[string]*EntryVar					// 1:1
}

type EntryUser struct {
	name string // KEY

	pw string
}

type EntryDelegation struct {
	targetName string // KEY

	issuerName string
	varName string
	right byte
}

type EntryVar struct {
	name string // KEY

	value string // direct assignment
	fieldValues map[string]string // multiple fields
}

func NewDatabase() *Database {
	db := Database{
		users: make(map[string]*EntryUser, 0),
		delegations: make(map[string][]*EntryDelegation, 0),
		vars: make(map[string]*EntryVar, 0),
	}
	return &db
}

// >>>>>>>>>>>>>>> QUERIES >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func (db *Database) isLoginCorrect(name, pw string) bool {
	u := db.users[name]
	return u != nil && u.pw == pw
}

func (db *Database) isUserExists(name string) bool {
	for k, _ := range db.users {
		if k == name {
			return true
		}
	}
	return false
}

func (db *Database) AddUser(name, pw string) {
	db.users[name] = &EntryUser{name: name, pw: pw}
}
