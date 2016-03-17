// Copyright 2015 Apcera Inc. All rights reserved.

package mockprovider

import (
	"net"

	libssh "github.com/apcera/libretto/ssh"
	lvm "github.com/apcera/libretto/virtualmachine"
)

// VM represents a Mock VM wrapper.
type VM struct {
	MockGetSSH    func(options libssh.Options) (libssh.Client, error)
	MockDestroy   func() error
	MockHalt      func() error
	MockSuspend   func() error
	MockResume    func() error
	MockStart     func() error
	MockGetIPs    func() ([]net.IP, error)
	MockGetName   func() string
	MockGetState  func() (string, error)
	MockProvision func() error
}

var _ lvm.VirtualMachine = (*VM)(nil)

// GetName returns the name of the virtual machine
func (vm *VM) GetName() string {
	if vm.MockGetName != nil {
		return vm.MockGetName()
	}
	return ""
}

// GetSSH returns an ssh client for the the vm.
func (vm *VM) GetSSH(options libssh.Options) (libssh.Client, error) {
	if vm.MockGetSSH != nil {
		return vm.MockGetSSH(options)
	}
	return nil, lvm.ErrNotImplemented
}

// Destroy powers off the VM and deletes its files from disk.
func (vm *VM) Destroy() error {
	if vm.MockDestroy != nil {
		return vm.MockDestroy()
	}
	return lvm.ErrNotImplemented
}

// Halt powers off the VM without destroying it
func (vm *VM) Halt() error {
	if vm.MockHalt != nil {
		return vm.MockHalt()
	}
	return lvm.ErrNotImplemented
}

// Suspend suspends the active state of the VM.
func (vm *VM) Suspend() error {
	if vm.MockSuspend != nil {
		return vm.MockSuspend()
	}
	return lvm.ErrNotImplemented
}

// Resume suspends the active state of the VM.
func (vm *VM) Resume() error {
	if vm.MockResume != nil {
		return vm.MockResume()
	}
	return lvm.ErrNotImplemented
}

// Start powers on the VM
func (vm *VM) Start() error {
	if vm.MockStart != nil {
		return vm.MockStart()
	}
	return lvm.ErrNotImplemented
}

// GetIPs returns a list of ip addresses associated with the vm through VMware tools
func (vm *VM) GetIPs() ([]net.IP, error) {
	if vm.MockGetIPs != nil {
		return vm.MockGetIPs()
	}
	return []net.IP{}, nil
}

// GetState gets the power state of the VM through VMware tools.
func (vm *VM) GetState() (string, error) {
	if vm.MockGetState != nil {
		return vm.MockGetState()
	}
	return "", lvm.ErrNotImplemented
}

// Provision clones this VM and powers it on, while waiting for it to get an IP address.
func (vm *VM) Provision() error {
	if vm.MockProvision != nil {
		return vm.MockProvision()
	}
	return lvm.ErrNotImplemented
}
