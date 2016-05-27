// Copyright 2016 Apcera Inc. All rights reserved.

package google

import (
	"errors"
	"net"
	"time"

	"github.com/apcera/libretto/ssh"
	"github.com/apcera/libretto/virtualmachine"
)

var (
	defaultZone        = "us-central1-a"
	defaultUser        = "ubuntu"
	defaultMachineType = "n1-standard-1"
	defaultSourceImage = "ubuntu-1404-trusty-v20160516"
	defaultScopes      = []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/compute",
		"https://www.googleapis.com/auth/devstorage.full_control",
		"https://www.googleapis.com/auth/logging.write",
		"https://www.googleapis.com/auth/monitoring.write",
		"https://www.googleapis.com/auth/servicecontrol",
		"https://www.googleapis.com/auth/service.management",
	}
	defaultDiskType = "pd-standard"
	defaultDiskSize = 10

	// Compiler will complain if google.VM doesn't implement VirtualMachine interface.
	_ virtualmachine.VirtualMachine = (*VM)(nil)
)

type VM struct {
	Name        string
	Zone        string
	MachineType string
	SourceImage string
	DiskType    string
	DiskSize    int // GB

	Preemptible   bool
	Network       string
	Subnetwork    string
	UseInternalIP bool

	Scopes  []string //Access scopes
	Project string   //GCE project
	Tags    []string //Instance Tags

	AccountFile  string
	account      credFile
	SSHCreds     ssh.Credentials // required
	SSHPublicKey string
}

const (
	// PublicIP is the index of the public IP address that GetIPs returns.
	PublicIP = 0
	// PrivateIP is the index of the private IP address that GetIPs returns.
	PrivateIP = 1

	// SSHTimeout is the maximum time to wait before failing to GetSSH.
	SSHTimeout = 3 * time.Minute
)

// GetName returns the name of the virtual machine
func (vm *VM) GetName() string {
	return vm.Name
}

// Provision creates a virtual machine on GCE. It returns an error if
// there was a problem during creation.
func (vm *VM) Provision() error {
	s, err := vm.getService()
	if err != nil {
		return err
	}

	return s.provision()
}

// GetIPs returns a slice of IP addresses assigned to the VM.
func (vm *VM) GetIPs() ([]net.IP, error) {
	s, err := vm.getService()
	if err != nil {
		return nil, err
	}

	return s.getIPs()
}

// Destroy deletes the VM on GCE.
func (vm *VM) Destroy() error {
	s, err := vm.getService()
	if err != nil {
		return err
	}

	return s.delete()
}

// GetStates retrieve the instance status.
func (vm *VM) GetState() (string, error) {
	s, err := vm.getService()
	if err != nil {
		return "", err
	}

	instance, err := s.getInstance()
	if err != nil {
		return "", err
	}

	return instance.Status, nil

}

// Don't support, return the error
func (vm *VM) Suspend() error {
	return errors.New("Suspend action not supported by GCE")
}

// Don't support, return the error
func (vm *VM) Resume() error {
	return errors.New("Resume action not supported by GCE")
}

// Halt stops a GCE instance
func (vm *VM) Halt() error {
	s, err := vm.getService()
	if err != nil {
		return err
	}

	return s.stop()
}

// Start a stopped GCE instance
func (vm *VM) Start() error {
	s, err := vm.getService()
	if err != nil {
		return err
	}

	return s.start()
}

// GetSSH returns an SSH client connected to the instance
func (vm *VM) GetSSH(options ssh.Options) (ssh.Client, error) {
	ips, err := vm.GetIPs()
	if err != nil {
		return nil, err
	}

	client := &ssh.SSHClient{
		Creds:   &vm.SSHCreds,
		IP:      ips[PublicIP],
		Options: options,
		Port:    22,
	}

	if err := client.WaitForSSH(SSHTimeout); err != nil {
		return nil, err
	}

	return client, nil
}
