// Copyright 2015 Apcera Inc. All rights reserved.

// +build windows, !amd64

package vmrun

import (
	"os"
	"path/filepath"
)

// Hardcoded path to vmrun to fallback when it is not on path.
var VMRunPath string

func init() {
	VMRunPath = filepath.Join(os.Getenv("ProgramFiles"), "VMware", "VMware VIX", "vmrun.exe")
}
