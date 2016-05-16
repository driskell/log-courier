package main

import "os/user"

// CredentialCache holds a cache of UID and GID unique identifiers
type CredentialCache struct {
	users  map[string]string
	groups map[string]string
}

// NewCredentialCache creates a new CredentialCache structure
func NewCredentialCache() *CredentialCache {
	return &CredentialCache{
		users:  map[string]string{},
		groups: map[string]string{},
	}
}

// LookupUser returns the unique UID for the given username, returning it from
// the cache if already looked up
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

// LookupGroup returns the unique GID for the given groupname, returning it from
// the cache if already looked up
func (c *CredentialCache) LookupGroup(groupname string) (string, error) {
	if gid, ok := c.groups[groupname]; ok {
		return gid, nil
	}

	// Since we need to support Golang < 1.7 use our backported LookupGroup
	lookup, err := LookupGroup(groupname)
	if err != nil {
		return "", err
	}

	c.groups[groupname] = lookup.Gid
	return lookup.Gid, nil
}
