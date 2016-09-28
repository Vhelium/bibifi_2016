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

const (
	VAR_MODE_SINGLE int = 0
	VAR_MODE_RECORD int = 1
	VAR_MODE_LIST int = 2
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
	list []*EntryVar // list
}

func NewDatabase() *Database {
	db := Database{
		users: make(map[string]*EntryUser, 0),
		delegations: make(map[string][]*EntryDelegation, 0),
		vars: make(map[string]*EntryVar, 0),
	}
	return &db
}

func SnapshotDatabase(env *GlobalEnv) {
	users := make(map[string]*EntryUser, len(env.db.users))
	delegations := make(map[string][]*EntryDelegation, len(env.db.delegations))
	vars := make(map[string]*EntryVar, len(env.db.vars))

	for k,v := range env.db.users {
		users[k] = &EntryUser{v.name, v.pw}
	}
	for k,v := range env.db.delegations {
		delegations[k] = make([]*EntryDelegation, len(v))
		copy(delegations[k], v)
	}
	for k,v := range env.db.vars {
		var fv map[string]string
		var l []*EntryVar
		if v.mode == VAR_MODE_RECORD {
			fv = make(map[string]string, len(v.fieldValues))
			for fk, f := range v.fieldValues {
				fv[fk] = f
			}
		} else if v.mode == VAR_MODE_LIST {
			// TODO: test this^^
			lst := make([]*EntryVar, len(v.list))
			for i, l := range v.list {
				if l.mode == VAR_MODE_RECORD {
					fv = make(map[string]string, len(l.fieldValues))
					for fk, f := range l.fieldValues {
						fv[fk] = f
					}
				}
				lst[i] = &EntryVar{
					name: "",
					mode: l.mode,
					value: l.value,
					fieldValues: fv,
				}
			}
		}
		vars[k] = &EntryVar{v.name, v.mode, v.value, fv, l}
	}
	env.dbSnapshot = &Database{users, delegations, vars}
}

func RollbackDatabase(env *GlobalEnv) {
	env.db = env.dbSnapshot
	env.dbSnapshot = nil
}

func NewEntryVar(ident string, val *Value) *EntryVar {
	var l []*EntryVar
	if val.mode == VAR_MODE_LIST {
		l = make([]*EntryVar, len(val.list))
		for i,v := range val.list {
			l[i] = NewEntryVar("", v)
		}
	}
	return &EntryVar {
		name: ident,
		mode: val.mode,
		value: val.val,
		fieldValues: val.vals,
		list: l,
	}
}

// >>>>>>>>>>>>>> MISC >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func (db *Database) printDB() {
	fmt.Printf(">>> DATABASE DUMP >>>\n")
	fmt.Printf("USERS:\n")
	for k, v := range db.users {
		fmt.Printf("\t{%s: %s-%s}\n", k, v.name, v.pw)
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
			fmt.Printf("\t{%s %s %s -> %s}\n", v.varName, v.issuerName,
				right, v.targetName)
		}
	}
	fmt.Printf("\nGLOBALS:\n")
	for k, v := range db.vars {
		fmt.Printf("\t{%s: %s}\n", k, printValue(v))
	}
	fmt.Printf("\n>>>>>>>>>>>>>>>>>>>>>\n")
}

func printValue(v *EntryVar) string {
	if v.mode == 0 {
		return v.value
	} else if v.mode == VAR_MODE_LIST{
		s := "["
		for i,l := range v.list {
			s += printValue(l)
			if i < len(v.list) - 1 {
				s += ", "
			}
		}
		return s + "]"
	} else {
		return fmt.Sprintf("%v", v.fieldValues)
	}
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

// check privileges before setting
func (db *Database) setGlobalVarFor(ident string, val *Value, user string) int {
	// check if variable exists && user has WRITE rights on it
	if db.doesGlobalVarExist(ident) {
		if db.hasUserPrivilege(ident, user, WRITE) {
			db.vars[ident] = NewEntryVar(ident, val)
			return DB_SUCCESS
		} else {
			//insufficient perms
			return DB_INSUFFICIENT_RIGHTS
		}
	} else {
		// otherwise, create new w/ corresponding rights
		db.vars[ident] = NewEntryVar(ident, val)
		db.setDelegationAllRights(ident, "admin", user)
		return DB_SUCCESS
	}
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

func (db *Database) setDelegationAllRights(varName, issuer, target string) int {
	//TODO: give all rights to that user
	return DB_SUCCESS
}

func (db *Database) removeDelegation(varName, issuer, target string,
		r AccessRight) int {
	//TODO: revoke right `r` from user
	return DB_SUCCESS
}

func (db *Database) setDelegationAllVars(issuer, target string, r AccessRight) int {
	//TODO: adds (zero or more) assertions of the form x i <right> -> t
	// for all variables x on which i has delegate permission
	return DB_SUCCESS
}

func (db *Database) removeDelegationAllVars(issuer, target string, r AccessRight) int {
	//TODO: revokes (zero or more) assertions of the form x i <right> -> t
	// for those variables x on which i has delegate permission
	return DB_SUCCESS
}

func (db *Database) hasUserPrivilege(varName, user string, r AccessRight) bool {
	// TODO: return true if user has right `r`
	return true
}
