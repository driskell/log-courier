package main

import (
  "log"
)

func (e *NewFileEvent) Process(state map[*ProspectorInfo]*FileState) {
  // A new file we need to save offset information for so we can resume
  state[e.ProspectorInfo] = &FileState{
    Source: &e.Source,
    Offset: e.Offset,
  }
  file_ids(e.fileinfo, state[e.ProspectorInfo])
}

func (e *DeletedEvent) Process(state map[*ProspectorInfo]*FileState) {
  // Purge the registrar entry - means the file is deleted so we can't resume
  // This keeps the state clean so it doesn't build up after thousands of log files
  delete(state, e.ProspectorInfo)
}

func (e *RenamedEvent) Process(state map[*ProspectorInfo]*FileState) {
  _, is_found := state[e.ProspectorInfo]
  if !is_found {
    // This is probably stdin or a deleted file we can't resume
    return
  }
  // Update the stored file name
  state[e.ProspectorInfo].Source = &e.Source
}

func (e *EventsEvent) Process(state map[*ProspectorInfo]*FileState) {
  if len(e.Events) == 1 {
    log.Printf("Registrar received %d event\n", len(e.Events))
  } else {
    log.Printf("Registrar received %d events\n", len(e.Events))
  }

  for _, event := range e.Events {
    _, is_found := state[event.ProspectorInfo]
    if !is_found {
      // This is probably stdin then or a deleted file we can't resume
      continue
    }

    state[event.ProspectorInfo].Offset = event.Offset
  }
}

func Registrar(state map[*ProspectorInfo]*FileState, registrar <-chan []RegistrarEvent) {
  for events := range registrar {
    for _, event := range events {
      event.Process(state)
    }

    state_json := make(map[string]*FileState, len(state))
    for _, value := range state {
      state_json[*value.Source] = value
    }

    WriteRegistry(state_json, ".logstash-forwarder")
  }
}
