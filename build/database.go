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
	USER_ADMIN  string = "admin"
	USER_ANYONE string = "anyone"
)

const (
	VAR_MODE_SINGLE int = 0
	VAR_MODE_RECORD int = 1
	VAR_MODE_LIST   int = 2
)

type AccessRight byte

const (
	READ     AccessRight = 1
	WRITE    AccessRight = 2
	APPEND   AccessRight = 4
	DELEGATE AccessRight = 8
)

type Database struct {
	defaultDelegator string
	principals       map[string]*EntryUser         // 1:1
	delegations      map[string][]*EntryDelegation // 1:N
	vars             map[string]*EntryVar          // 1:1
}

type EntryUser struct {
	name string // KEY

	pw string
}

type EntryDelegation struct {
	targetName string // KEY

	issuerName string
	varName    string
	right      AccessRight
}

type EntryVar struct {
	name string // KEY

	mode        int               // 0 = direct, 1 = fields
	value       string            // direct assignment
	fieldValues map[string]string // multiple fields
	list        []*EntryVar       // list
}

func NewDatabase() *Database {
	db := &Database{
		defaultDelegator: USER_ANYONE,
		principals:       make(map[string]*EntryUser, 0),
		delegations:      make(map[string][]*EntryDelegation, 0),
		vars:             make(map[string]*EntryVar, 0),
	}
	db.defaultDelegator = USER_ANYONE
	db.principals[USER_ADMIN] = &EntryUser{name: USER_ADMIN, pw: "admin"}
	return db
}

func SnapshotDatabase(env *GlobalEnv) {
	principals := make(map[string]*EntryUser, len(env.db.principals))
	delegations := make(map[string][]*EntryDelegation, len(env.db.delegations))
	vars := make(map[string]*EntryVar, len(env.db.vars))

	for k, v := range env.db.principals {
		principals[k] = &EntryUser{v.name, v.pw}
	}
	for k, v := range env.db.delegations {
		delegations[k] = make([]*EntryDelegation, len(v))
		copy(delegations[k], v)
	}
	for k, v := range env.db.vars {
		var fv map[string]string
		var lst []*EntryVar
		if v.mode == VAR_MODE_RECORD {
			fv = make(map[string]string, len(v.fieldValues))
			for fk, f := range v.fieldValues {
				fv[fk] = f
			}
		} else if v.mode == VAR_MODE_LIST {
			lst = make([]*EntryVar, len(v.list))
			for i, l := range v.list {
				if l.mode == VAR_MODE_RECORD {
					fv = make(map[string]string, len(l.fieldValues))
					for fk, f := range l.fieldValues {
						fv[fk] = f
					}
				}
				lst[i] = &EntryVar{
					name:        "",
					mode:        l.mode,
					value:       l.value,
					fieldValues: fv,
				}
			}
		}
		vars[k] = &EntryVar{v.name, v.mode, v.value, fv, lst}
	}
	env.dbSnapshot = &Database{env.db.defaultDelegator, principals, delegations, vars}
}

func RollbackDatabase(env *GlobalEnv) {
	env.db = env.dbSnapshot
	env.dbSnapshot = nil
}

func NewEntryVar(ident string, val *Value) *EntryVar {
	var l []*EntryVar
	if val.mode == VAR_MODE_LIST {
		l = make([]*EntryVar, len(val.list))
		for i, v := range val.list {
			l[i] = NewEntryVar("", v)
		}
	}
	return &EntryVar{
		name:        ident,
		mode:        val.mode,
		value:       val.val,
		fieldValues: val.vals,
		list:        l,
	}
}

func NewValue(ev *EntryVar) *Value {
	var l []*Value
	if ev.mode == VAR_MODE_LIST {
		l = make([]*Value, len(ev.list))
		for i, v := range ev.list {
			l[i] = NewValue(v)
		}
	}
	return &Value{
		mode: ev.mode,
		val:  ev.value,
		vals: ev.fieldValues,
		list: l,
	}
}

// >>>>>>>>>>>>>> MISC >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func (env *ProgramEnv) printDB() {
	db := env.globals.db
	fmt.Printf(">>> DATABASE DUMP >>>\n")
	fmt.Printf("USERS:\n")
	for k, v := range db.principals {
		fmt.Printf("\t{%s: %s-%s}\n", k, v.name, v.pw)
	}
	fmt.Printf("\nDELEGATIONS:\n")
	for _, d := range db.delegations {
		for _, v := range d {
			right := "N/A"
			switch v.right {
			case READ:
				right = "READ"
			case WRITE:
				right = "WRITE"
			case APPEND:
				right = "APPEND"
			case DELEGATE:
				right = "DELEGATE"
			}
			fmt.Printf("\t{%s %s %s -> %s}\n", v.varName, v.issuerName,
				right, v.targetName)
		}
	}
	fmt.Printf("\nGLOBALS:\n")
	for k, v := range db.vars {
		fmt.Printf("\t{%s: %s}\n", k, printValue(v))
	}
	fmt.Printf("\nLOCALS:\n")
	for k, v := range env.locals {
		fmt.Printf("\t{%s: %s}\n", k, printValue(v))
	}
	fmt.Printf("\n>>>>>>>>>>>>>>>>>>>>>\n")
}

func printValue(v *EntryVar) string {
	if v.mode == 0 {
		return v.value
	} else if v.mode == VAR_MODE_LIST {
		s := "["
		for i, l := range v.list {
			s += printValue(l)
			if i < len(v.list)-1 {
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
	u := db.principals[name]
	return u != nil && u.pw == pw
}

func (db *Database) isUserExists(name string) bool {
	for k, _ := range db.principals {
		if k == name {
			return true
		}
	}
	return false
}

func (env *ProgramEnv) addUser(name, pw string) {
	env.globals.db.principals[name] = &EntryUser{name: name, pw: pw}
	// give default permissions via default delegator
	for _, r := range []AccessRight{READ, WRITE, DELEGATE, APPEND} {
		env.setDelegationAllVars(env.globals.db.defaultDelegator, name, r)
	}
}

func (db *Database) changePassword(name, pw string) {
	db.principals[name].pw = pw
}

func (env *ProgramEnv) doesUserExist(name string) bool {
	_, ok := env.globals.db.principals[name]
	return ok
}

func (db *Database) isUserAdmin(name string) bool {
	return name == USER_ADMIN
}

func (env *ProgramEnv) getVarValueForWith(ident, principal string,
	rs ...AccessRight) (int, *Value) {
	// check locals
	if env.doesLocalVarExist(ident) {
		return DB_VAR_FOUND, NewValue(env.getLocalVar(ident))
	}

	// check globals
	if !env.hasUserPrivilegeAtLeastOne(ident, principal, rs...) {
		return DB_INSUFFICIENT_RIGHTS, nil
	}
	if ev, ok := env.globals.db.vars[ident]; ok {
		return DB_VAR_FOUND, NewValue(ev)
	}
	return DB_VAR_NOT_FOUND, nil
}

func (env *ProgramEnv) getLocalVar(ident string) *EntryVar {
	return env.locals[ident]
}

// creates a new local variable (w/ LOCAL command)
func (env *ProgramEnv) setLocalVar(ident string, val *Value) int {
	// check if variable already exists (global/locals)
	if env.doesGlobalVarExist(ident) ||
		env.doesLocalVarExist(ident) {
		return DB_VAR_NOT_FOUND
	}
	env.locals[ident] = NewEntryVar(ident, val)
	return DB_SUCCESS
}

func (env *ProgramEnv) discardLocalVar(ident string) {
	delete(env.locals, ident)
}

func (env *ProgramEnv) doesLocalVarExist(ident string) bool {
	_, ok := env.locals[ident]
	return ok
}

func (env *ProgramEnv) doesVarExist(ident string) bool {
	return env.doesLocalVarExist(ident) || env.doesGlobalVarExist(ident)
}

func (env *ProgramEnv) setVarForWith(ident string, val *Value, principal string,
	rs ...AccessRight) int {
	db := env.globals.db
	// check locals
	if env.doesLocalVarExist(ident) {
		env.locals[ident] = NewEntryVar(ident, val)
		return DB_SUCCESS
	}
	// check if variable exists && principal has `rs` rights on it
	if env.doesGlobalVarExist(ident) {
		if env.hasUserPrivilegeAtLeastOne(ident, principal, rs...) {
			db.vars[ident] = NewEntryVar(ident, val)
			return DB_SUCCESS
		} else {
			//insufficient perms
			return DB_INSUFFICIENT_RIGHTS
		}
	} else {
		// otherwise, create new w/ corresponding rights
		db.vars[ident] = NewEntryVar(ident, val)
		env.setDelegationAllRights(ident, principal, principal)
		return DB_SUCCESS
	}
}

func (env *ProgramEnv) doesGlobalVarExist(ident string) bool {
	_, ok := env.globals.db.vars[ident]
	return ok
}

func (env *ProgramEnv) getFieldValueForWith(ident, field, principal string,
	rs ...AccessRight) (int, string) {
	db := env.globals.db
	if !env.hasUserPrivilegeAtLeastOne(ident, principal, rs...) {
		return DB_INSUFFICIENT_RIGHTS, ""
	}
	var ev *EntryVar
	var ok bool
	if env.doesLocalVarExist(ident) {
		ev = env.getLocalVar(ident)
		ok = true
	} else {
		ev, ok = db.vars[ident]
	}
	if ok && ev.mode == 1 {
		if f, ok := ev.fieldValues[field]; ok {
			return DB_VAR_FOUND, f
		}
	}
	return DB_VAR_NOT_FOUND, ""
}

// ident must be an existing list w/ needed rights(write, append)
func (env *ProgramEnv) appendVarToListFor(ident string, val *Value, pr string) int {
	if val.mode != VAR_MODE_SINGLE && val.mode != VAR_MODE_RECORD {
		return DB_VAR_NOT_FOUND
	}
	s, l := env.getVarValueForWith(ident, pr)
	if s == DB_VAR_FOUND {
		l.list = append(l.list, val)
		env.setVarForWith(ident, l, pr) // we already know we have the rights
		return DB_SUCCESS
	} else {
		return s
	}
}

// ident must be an existing list w/ needed rights(write, append)
func (env *ProgramEnv) concatListToListFor(ident string, val *Value, pr string) int {
	if val.mode != VAR_MODE_LIST {
		return DB_VAR_NOT_FOUND
	}
	s, l := env.getVarValueForWith(ident, pr)
	if s == DB_VAR_FOUND {
		for _, item := range val.list {
			l.list = append(l.list, item)
		}
		env.setVarForWith(ident, l, pr)
		return DB_SUCCESS
	} else {
		return s
	}
}

// >>>>>>>>>>>>>>> DELEGATION ASSERTIONS >>>>>>>>>>>>>>>>>>>>>>>>>>

func (env *ProgramEnv) setDefaultDelegator(target string) {
	// rights have to be checked by caller.
	// TODO: implement logic for this..
	env.globals.db.defaultDelegator = target
}

func (env *ProgramEnv) getDelegationIndex(varName, issuer, target string,
		r AccessRight) (int, bool) {
	if delegs, ok := env.globals.db.delegations[target]; ok {
		for i, d := range delegs {
			if d.issuerName == issuer && d.varName == varName && d.right == r {
				return i, true
			}
		}
	}
	return -1, false
}


func (env *ProgramEnv) setDelegation(varName, issuer, target string,
		r AccessRight) int {
	db := env.globals.db

	if env.globals.db.isUserAdmin(target) {
		return DB_SUCCESS
	}

	// Fail #1: if either p or q does not exist
	_, issuerExists := db.principals[issuer]
	_, targetExists := db.principals[target]

	if !issuerExists || !targetExists {
		return DB_VAR_NOT_FOUND
	}

	// Fail #2 x does not exist or is local var
	if !env.doesGlobalVarExist(varName) {
		return DB_VAR_NOT_FOUND
	}

	// Fail #3: if q does not have delegate permission on varName
	hasDelegRight := false
	if delegs, ok := db.delegations[issuer]; ok {
		for _, d := range delegs {
			if d.varName == varName && d.right == DELEGATE {
				hasDelegRight = true
				break
			}
		}
	}
	if !db.isUserAdmin(env.principal) && !(env.principal == issuer) && !hasDelegRight {
		return DB_INSUFFICIENT_RIGHTS
	}

	entryDelegation := EntryDelegation{
		targetName: target,
		issuerName: issuer,
		varName:    varName,
		right:      r,
	}
	// check if this delegation already exists:
	_, exist := env.getDelegationIndex(varName, issuer, target, r)
	if !exist {
		db.delegations[target] = append(db.delegations[target], &entryDelegation)
	}

	return DB_SUCCESS
}

func (env *ProgramEnv) setDelegationAllRights(varName, issuer, target string) int {
	//TODO make more efficient lol
	for _, r := range []AccessRight{READ, WRITE, APPEND, DELEGATE} {
		env.setDelegation(varName, issuer, target, r)
	}
	return DB_SUCCESS
}

func (env *ProgramEnv) deleteDelegation(varName, issuer, target string,
	r AccessRight) int {
	db := env.globals.db

	if env.globals.db.isUserAdmin(target) {
		return DB_SUCCESS
	}

	// Fail #1: if either p or q does not exist
	_, issuerExists := db.principals[issuer]
	_, targetExists := db.principals[target]

	if !issuerExists || !targetExists {
		return DB_VAR_NOT_FOUND
	}

	// Fail #2 x does not exist or is local var
	if !env.doesGlobalVarExist(varName) {
		return DB_VAR_NOT_FOUND
	}

	// Fail #3: if q does not have delegate permission on varName
	hasDelegRight := false
	if delegs, ok := db.delegations[issuer]; ok {
		for _, d := range delegs {
			if d.varName == varName && d.right == DELEGATE {
				hasDelegRight = true
				break
			}
		}
	}
	if !(env.principal == target) && !db.isUserAdmin(env.principal) && !hasDelegRight {
		return DB_INSUFFICIENT_RIGHTS
	}

	i, ok := env.getDelegationIndex(varName, issuer, target, r)
	if ok {
		db.delegations[target] = append(db.delegations[target][:i],
			db.delegations[target][i+1:]...)
		return DB_SUCCESS
	}
	// not found, return success still, lol
	return DB_SUCCESS
}

func (env *ProgramEnv) setDelegationAllVars(issuer, target string, r AccessRight) int {
	// get all vars where ISSUER has right `r` on
	vars := make([]string, 0)
	if delegs, ok := env.globals.db.delegations[issuer]; ok {
		for _, d := range delegs {
			if d.right == r {
				vars = append(vars, d.varName)
			}
		}
	}
	// add those to `target`
	for _, v := range vars {
		s := env.setDelegation(v, issuer, target, r)
		if s != DB_SUCCESS {
			return s
		}
	}
	return DB_SUCCESS
}

func (env *ProgramEnv) removeDelegationAllVars(issuer, target string,
		r AccessRight) int {
	// get all vars where ISSUER has right `r` on
	vars := make([]string, 0)
	if delegs, ok := env.globals.db.delegations[issuer]; ok {
		for _, d := range delegs {
			if d.right == r {
				vars = append(vars, d.varName)
			}
		}
	}
	// remove those from `target`
	for _, v := range vars {
		s := env.deleteDelegation(v, issuer, target, r)
		if s != DB_SUCCESS {
			return s
		}
	}
	return DB_SUCCESS
}

func (env *ProgramEnv) hasUserPrivilege(varName, principal string, r AccessRight) bool {
	return env.hasUserPrivilegeAtLeastOne(varName, principal, r)
}

func (env *ProgramEnv) hasUserPrivilegeAtLeastOne(varName, principal string,
	rs ...AccessRight) bool {
	if env.globals.db.isUserAdmin(principal) || env.doesLocalVarExist(varName) {
		return true
	}
	for _,p := range []string{principal, USER_ANYONE} {
		if delegs, ok := env.globals.db.delegations[p]; ok {
			// loop all delegation statements for that principal
			for _, deleg := range delegs {
				if deleg.varName == varName {
					// loop all possible rights
					for _, r := range rs {
						if deleg.right == r {
							return true
						}
					}
				}
			}
		}
	}
	return false
}
