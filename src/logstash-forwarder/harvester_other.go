// +build !windows

package main

import (
  "os"
)

func (h *Harvester) openFile(path string) (*os.File, error) {
  return os.Open(path)
}
