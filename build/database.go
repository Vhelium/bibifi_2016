package main

import (
	"fmt"
)

const (
	DB_VAR_NOT_FOUND int = iota
	DB_VAR_FOUND
	DB_SUCCESS
	DB_INSUFFICIENT_RIGHTS
)

type AccessRight byte
const (
	READ AccessRight = 1
	WRITE AccessRight = 2
	APPEND AccessRight = 4
	DELEGATE AccessRight = 8
)

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
	right AccessRight
}

type EntryVar struct {
	name string // KEY

	mode int // 0 = direct, 1 = fields
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

func NewEntryVar(ident string, val *Value) *EntryVar {
	return &EntryVar {
		name: ident,
		mode: val.s,
		value: val.val,
		fieldValues: val.vals,
	}
}

// >>>>>>>>>>>>>> MISC >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func (db *Database) printDB() {
	fmt.Printf(">>> DATABASE DUMP >>>\n")
	fmt.Printf("USERS:\n")
	for k, v := range db.users {
		fmt.Printf("\t{%s: %s-%s}, ", k, v.name, v.pw)
	}
	fmt.Printf("\nDELEGATIONS:\n")
	for _, d := range db.delegations {
		for _, v := range d {
			right := "N/A"
			switch v.right {
			case READ: right = "READ"
			case WRITE: right = "WRITE"
			case APPEND: right = "APPEND"
			case DELEGATE: right = "DELEGATE"
			}
			fmt.Printf("\t{%s %s %s -> %s}, ", v.varName, v.issuerName,
				right, v.targetName)
		}
	}
	fmt.Printf("\nGLOBALS:\n")
	for k, v := range db.vars {
		fmt.Printf("\t{%s: %s/%v}, ", k, v.value, v.fieldValues)
	}
	fmt.Printf("\n>>>>>>>>>>>>>>>>>>>>>\n")
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

func (db *Database) addUser(name, pw string) {
	db.users[name] = &EntryUser{name: name, pw: pw}
}

func (db *Database) getVarValueFor(ident, user string) (int, string) {
	if !db.hasUserPrivilege(ident, user, READ) {
		return DB_INSUFFICIENT_RIGHTS, ""
	}
	if ev, ok := db.vars[ident]; ok && ev.mode == 0 {
		return DB_VAR_FOUND, ev.value
	}
	return DB_VAR_NOT_FOUND, ""
}

func (db *Database) setGlobalVar(ident string, val *Value) {
	db.vars[ident] = NewEntryVar(ident, val)
}

func (db *Database) doesGlobalVarExist(ident string) bool {
	_, ok := db.vars[ident]
	return ok
}

func (db *Database) getFieldValueFor(ident, field, user string) (int, string) {
	if !db.hasUserPrivilege(ident, user, READ) {
		return DB_INSUFFICIENT_RIGHTS, ""
	}
	if ev, ok := db.vars[ident]; ok && ev.mode == 1 {
		if f, ok := ev.fieldValues[field]; ok {
			return DB_VAR_FOUND, f
		}
	}
	return DB_VAR_NOT_FOUND, ""
}

// >>>>>>>>>>>>>>> DELEGATION ASSERTIONS >>>>>>>>>>>>>>>>>>>>>>>>>>

func (db *Database) setDelegation(varName, issuer, target string, r AccessRight) int {
	//TODO: give right `r` to that user
	return DB_SUCCESS
}

func (db *Database) setAllDelegations(varName, issuer, target string) int {
	//TODO: give all rights to that user
	return DB_SUCCESS
}

func (db *Database) removeDelegation(varName, issuer, target string,
		r AccessRight) int {
	//TODO: revoke right `r` from user
	return DB_SUCCESS
}

func (db *Database) hasUserPrivilege(varName, user string, r AccessRight) bool {
	// TODO: return true if user has right `r`
	return true
}
