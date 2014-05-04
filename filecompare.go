package main

import (
  "os"
)

func lookup_file_ids(file string, info os.FileInfo, fileinfo map[string]*ProspectorInfo, missingfiles map[string]*ProspectorInfo) (string, *ProspectorInfo) {
  for kf, ki := range fileinfo {
    if kf == file {
      continue
    }
    if os.SameFile(ki.fileinfo, info) {
      return kf, ki
    }
  }

  // Now check the missingfiles
  for kf, ki := range missingfiles {
    if os.SameFile(ki.fileinfo, info) {
      return kf, ki
    }
  }
  return "", nil
}

func lookup_file_ids_resume(file string, info os.FileInfo, initial map[string]*FileState) string {
  for kf, ki := range initial {
    if kf == file {
      continue
    }
    if is_filestate_same(file, info, ki) {
      return kf
    }
  }

  return ""
}
