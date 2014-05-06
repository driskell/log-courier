package main

import "os"

type FileEvent struct {
  ProspectorInfo *ProspectorInfo
  Source         *string
  Offset         int64
  Line           uint64
  Text           *string
  Fields         *map[string]string
}

type RegistrarEvent interface {
  Process(state map[*ProspectorInfo]*FileState)
}

type NewFileEvent struct {
  ProspectorInfo *ProspectorInfo
  Source         string
  Offset         int64
  fileinfo       *os.FileInfo
}

type DeletedEvent struct {
  ProspectorInfo *ProspectorInfo
}

type RenamedEvent struct {
  ProspectorInfo *ProspectorInfo
  Source         string
}

type EventsEvent struct {
  Events []*FileEvent
}
