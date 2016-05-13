// +build darwin dragonfly freebsd !android,linux netbsd openbsd solaris

package main

import (
	"os/exec"
	"strconv"
)

func (m *MuninRunner) applyCommandUser(command *exec.Cmd, username string, cache *CredentialCache) error {
	uid, err := cache.LookupUser(username)
	if err != nil {
		return err
	}

	uidInt, _ := strconv.Atoi(uid)
	command.SysProcAttr.Credential.Uid = uint32(uidInt)

	return nil
}

func (m *MuninRunner) applyCommandGroup(command *exec.Cmd, groupname string, cache *CredentialCache) error {
	gid, err := cache.LookupGroup(groupname)
	if err != nil {
		return err
	}

	gidInt, _ := strconv.Atoi(gid)
	command.SysProcAttr.Credential.Gid = uint32(gidInt)

	return nil
}
