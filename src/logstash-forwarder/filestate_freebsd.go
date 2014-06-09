package main

import (
  "os"
  "syscall"
)

type FileStateOS struct {
  Inode  uint32 `json:"inode,omitempty"`
  Device uint32 `json:"device,omitempty"`
}

func (fs *FileStateOS) PopulateFileIds(info os.FileInfo) {
  fstat := info.Sys().(*syscall.Stat_t)
  fs.Inode = fstat.Ino
  fs.Device = fstat.Dev
}

func (fs *FileStateOS) SameAs(info os.FileInfo) bool {
  state := &FileStateOS{}
  state.PopulateFileIds(info)
  return (fs.Inode == state.Inode && fs.Device == state.Device)
}
