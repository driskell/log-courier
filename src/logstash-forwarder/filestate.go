package main

import (
  "os"
)

type FileState struct {
  FileStateOS
  Source *string `json:"source,omitempty"`
  Offset int64   `json:"offset,omitempty"`
}

type FileInfo struct {
  fileinfo os.FileInfo /* the file info */
}

func NewFileInfo(fileinfo os.FileInfo) *FileInfo {
  return &FileInfo{
    fileinfo: fileinfo,
  }
}

func (fs *FileInfo) SameAs(info os.FileInfo) bool {
  return os.SameFile(info, fs.fileinfo)
}

func (fs *FileInfo) Stat() os.FileInfo {
  return fs.fileinfo
}

func (fs *FileInfo) Update(fileinfo os.FileInfo, identity *FileIdentity) {
  fs.fileinfo = fileinfo
}

func (fs *FileState) Stat() os.FileInfo {
  return nil
}

func (fs *FileState) Update(fileinfo os.FileInfo, identity *FileIdentity) {
  // Promote to a FileInfo
  (*identity) = NewFileInfo(fileinfo)
}

type FileIdentity interface {
  SameAs(os.FileInfo) bool
  Stat() os.FileInfo
  Update(os.FileInfo, *FileIdentity)
}
