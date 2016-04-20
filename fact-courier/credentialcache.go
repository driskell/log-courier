package main

import "os/user"

type CredentialCache struct {
	users  map[string]string
	groups map[string]string
}

func NewCredentialCache() *CredentialCache {
	return &CredentialCache{
		users:  map[string]string{},
		groups: map[string]string{},
	}
}

func (c *CredentialCache) LookupUser(username string) (string, error) {
	if uid, ok := c.users[username]; ok {
		return uid, nil
	}

	lookup, err := user.Lookup(username)
	if err != nil {
		return "", err
	}

	c.users[username] = lookup.Uid
	return lookup.Uid, nil
}

func (c *CredentialCache) LookupGroup(groupname string) (string, error) {
	if gid, ok := c.groups[groupname]; ok {
		return gid, nil
	}

	lookup, err := LookupGroup(groupname)
	if err != nil {
		return "", err
	}

	c.groups[groupname] = lookup.Gid
	return lookup.Gid, nil
}
