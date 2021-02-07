// +build darwin dragonfly freebsd !android,linux netbsd openbsd solaris

package main

import (
	"os/exec"
	"strconv"
)

// applyCommandUser sets the run user for the given command
func (m *MuninRunner) applyCommandUser(command *exec.Cmd, username string, cache *CredentialCache) error {
	uid, err := cache.LookupUser(username)
	if err != nil {
		return err
	}

	uidInt, _ := strconv.ParseUint(uid, 10, 32)
	command.SysProcAttr.Credential.Uid = uint32(uidInt)

	log.Debug("[%s] Uid: %d", m.name, uint32(uidInt))

	return nil
}

// applyCommandGroup sets the run group for the given command
func (m *MuninRunner) applyCommandGroup(command *exec.Cmd, groupname string, cache *CredentialCache) error {
	gid, err := cache.LookupGroup(groupname)
	if err != nil {
		return err
	}

	gidInt, _ := strconv.ParseUint(gid, 10, 32)
	command.SysProcAttr.Credential.Gid = uint32(gidInt)

	log.Debug("[%s] Gid: %d", m.name, uint32(gidInt))

	return nil
}
