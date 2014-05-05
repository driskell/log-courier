package main

import (
  "log"
  "os"
)

const EVENT_NEWFILE = 0x00000001
const EVENT_DELETED = 0x00000002
const EVENT_RENAMED = 0x00000003
const EVENT_OFFSET = 0x00000004

type RegistrarEvent struct {
  ProspectorInfo *ProspectorInfo
  Type           uint32
  Source         string
  Offset         int64
  Events         uint32
  fileinfo       *os.FileInfo
}

func Registrar(state map[*ProspectorInfo]*FileState, registrar chan []*RegistrarEvent) {
  for registrar_events := range registrar {
    for _, event := range registrar_events {
      if event.Type == EVENT_NEWFILE {
        // A new file we need to save offset information for so we can resume
        state[event.ProspectorInfo] = &FileState{
          Source: &event.Source,
          Offset: event.Offset,
        }

        file_ids(event.fileinfo, state[event.ProspectorInfo])
      } else if event.Type == EVENT_DELETED {
        // Purge the registrar entry - means the file is deleted so we can't resume
        // This keeps the state clean so it doesn't build up after thousands of log files
        delete(state, event.ProspectorInfo)
      } else if event.Type == EVENT_RENAMED {
        _, is_found := state[event.ProspectorInfo]
        if !is_found {
          // This is probably stdin then or a deleted file we can't resume
          continue
        }

        state[event.ProspectorInfo].Source = &event.Source

        file_ids(event.fileinfo, state[event.ProspectorInfo])
      } else if event.Type == EVENT_OFFSET {
        _, is_found := state[event.ProspectorInfo]
        if !is_found {
          // This is probably stdin then or a deleted file we can't resume
          continue
        }

        if event.Events == 1 {
          log.Printf("Registrar received %d event\n", event.Events)
        } else {
          log.Printf("Registrar received %d events\n", event.Events)
        }

        state[event.ProspectorInfo].Offset = event.Offset

        file_ids(event.fileinfo, state[event.ProspectorInfo])
      }
    }

    state_json := make(map[string]*FileState, len(state))
    for _, value := range state {
      state_json[*value.Source] = value
    }

    WriteRegistry(state_json, ".logstash-forwarder")
  }
}
