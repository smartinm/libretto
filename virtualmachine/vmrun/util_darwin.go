// Copyright 2015 Apcera Inc. All rights reserved.

// +build darwin

package vmrun

import (
	"bytes"
	"os/exec"
)

func runVMware() (string, string, error) {
	// Open Fusion in case it is not already running
	cmd := exec.Command("open", "/Applications/VMware Fusion.app")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
