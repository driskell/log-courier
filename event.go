package main

import "os"

type FileEvent struct {
  ProspectorInfo *ProspectorInfo
  Source         *string
  Offset         int64
  Line           uint64
  Text           *string
  Fields         *map[string]string
  fileinfo       *os.FileInfo
}
