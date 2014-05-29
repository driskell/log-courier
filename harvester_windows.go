package main

import (
  "os"
)

func (h *Harvester) openFile(path string) (*os.File, error) {
  // We will call CreateFile directly so we can pass in FILE_SHARE_DELETE
  // This ensures that a program can still rotate the file even though we have it open
  pathp, err := syscall.UTF16PtrFromString(path)
  if err != nil {
    return nil, err
  }

  var sa *syscall.SecurityAttributes

  handle, err := syscall.CreateFile(
    pathp, syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
    sa, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
  if err != nil {
    return nil, err
  }

  return os.NewFile(uintptr(handle), path), nil
}
