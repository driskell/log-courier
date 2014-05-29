package main

import (
  "os"
  "syscall"
)

type FileState struct {
  Source *string `json:"source,omitempty"`
  Offset int64   `json:"offset,omitempty"`
  Inode  uint32  `json:"inode,omitempty"`
  Device uint32  `json:"device,omitempty"`
}

func file_ids(info os.FileInfo, state *FileState) {
  fstat := info.Sys().(*syscall.Stat_t)
  state.Inode = fstat.Ino
  state.Device = fstat.Dev
}

func (fs *FileState) SameAs(info os.FileInfo) bool {
  state := &FileState{}
  file_ids(info, state)
  return (fs.Inode == state.Inode && fs.Device == state.Device)
}
