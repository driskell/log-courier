// +build darwin dragonfly freebsd !android,linux netbsd openbsd solaris

package main

import "strconv"

func (m *MuninRunner) applyCommandUser(username string, cache *CredentialCache) error {
	uid, err := cache.LookupUser(username)
	if err != nil {
		return err
	}

	uidInt, _ := strconv.Atoi(uid)
	m.command.SysProcAttr.Credential.Uid = uint32(uidInt)

	return nil
}

func (m *MuninRunner) applyCommandGroup(groupname string, cache *CredentialCache) error {
	gid, err := cache.LookupGroup(groupname)
	if err != nil {
		return err
	}

	gidInt, _ := strconv.Atoi(gid)
	m.command.SysProcAttr.Credential.Gid = uint32(gidInt)

	return nil
}
