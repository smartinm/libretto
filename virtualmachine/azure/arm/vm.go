// Copyright 2016 Apcera Inc. All rights reserved.

// Package arm provides methods for creating and manipulating VMs on Azure using arm API.
package arm

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/apcera/libretto/ssh"
	"github.com/apcera/libretto/util"
	lvm "github.com/apcera/libretto/virtualmachine"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/go-autorest/autorest/azure"
)

const (
	// PublicIP is the index of the public IP address that GetIPs returns.
	PublicIP = 0

	// PrivateIP is the index of the private IP address that GetIPs returns.
	PrivateIP = 1

	// DefaultTimeout is the maximum seconds to wait before failing to GetSSH.
	defaultTimeout = 60

	// Running is status returned when the VM is running
	running = "VM running"
	// Stopped is status returned when the VM is halted
	stopped = "VM stopped"

	// Succeded is the status returned when a deployment ends successfully
	succeeded = "Succeeded"

	// Deployment name when provisioning a VM
	deploymentName = "libretto"

	// Maximum length that public ip can have
	maxPublicIPLength = 63
)

var _ lvm.VirtualMachine = (*VM)(nil)

// OAuthCredentials is the struct that stors OAUTH credentials
type OAuthCredentials struct {
	ClientID       string
	ClientSecret   string
	TenantID       string
	SubscriptionID string
}

// VM represents an Azure virtual machine.
type VM struct {
	// Credentials to connect Azure
	Creds OAuthCredentials

	// Image Properties
	ImagePublisher string
	ImageOffer     string
	ImageSku       string

	// VM Properties
	Size string
	Name string

	// SSH Properties
	SSHCreds ssh.Credentials // required

	// Deployment Properties
	ResourceGroup    string
	StorageAccount   string
	StorageContainer string

	// Authorizer to connect to Azure
	Authorizer *azure.ServicePrincipalToken

	// VM OS Properties
	OsFile string

	// VM Network Properties
	NetworkSecurityGroup string
	Nic                  string
	PublicIP             string
	Subnet               string
	VirtualNetwork       string
}

// GetName returns the name of the VM.
func (vm *VM) GetName() string {
	return vm.Name
}

// Provision creates a new VM instance on Azure. It returns an error if there
// was a problem during creation.
func (vm *VM) Provision() error {
	// Validate VM
	err := validateVM(vm)
	if err != nil {
		return err
	}

	// Set up the authorizer
	spt, err := NewServicePrincipalTokenFromCredentials(&vm.Creds, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return err
	}
	vm.Authorizer = spt

	// Set up private members of the VM
	tempName := fmt.Sprintf("%s-%s", vm.Name, randStringRunes(6))
	vm.OsFile = tempName + "-os-disk.vhd"
	vm.PublicIP = tempName + "-public-ip"
	vm.Nic = tempName + "-nic"

	publicIPLength := len(vm.PublicIP)
	if publicIPLength > maxPublicIPLength {
		vm.PublicIP = vm.PublicIP[publicIPLength-maxPublicIPLength:]
	}

	// Create and send the deployment
	vm.deploy()

	// Use GetSSH to try to connect to machine
	cli, err := vm.GetSSH(ssh.Options{KeepAlive: 2})
	if err != nil {
		return err
	}

	return cli.WaitForSSH(defaultTimeout * time.Second)
}

// GetIPs returns the IP addresses of the Azure VM instance.
func (vm *VM) GetIPs() ([]net.IP, error) {
	ips := make([]net.IP, 2)

	// Get the Public IP
	ip, err := vm.getPublicIP()
	if err != nil {
		return nil, err
	}
	ips[PublicIP] = ip

	// Get the Private IP
	ip, err = vm.getPrivateIP()
	if err != nil {
		return nil, err
	}
	ips[PrivateIP] = ip

	return ips, nil
}

// GetSSH returns an SSH client that can be used to connect to the VM. An error
// is returned if the VM has no IPs.
func (vm *VM) GetSSH(options ssh.Options) (ssh.Client, error) {
	ips, err := util.GetVMIPs(vm, options)
	if err != nil {
		return nil, err
	}

	client := ssh.SSHClient{
		Creds:   &vm.SSHCreds,
		IP:      ips[PublicIP],
		Options: options,
		Port:    22,
	}
	return &client, nil
}

// GetState returns the status of the Azure VM. The status will be one of the
// following:
//     "running"
//     "stopped"
func (vm *VM) GetState() (string, error) {
	virtualMachinesClient := compute.NewVirtualMachinesClient(vm.Creds.SubscriptionID)
	virtualMachinesClient.Authorizer = vm.Authorizer

	r, e := virtualMachinesClient.Get(vm.ResourceGroup, vm.Name, "InstanceView")
	if r.Properties != nil && r.Properties.InstanceView != nil {
		state := *(*r.Properties.InstanceView.Statuses)[1].DisplayStatus
		return translateState(state), e
	}
	return "", e
}

// Destroy deletes the VM on Azure.
func (vm *VM) Destroy() error {
	// Delete the VM
	virtualMachinesClient := compute.NewVirtualMachinesClient(vm.Creds.SubscriptionID)
	virtualMachinesClient.Authorizer = vm.Authorizer

	_, err := virtualMachinesClient.Delete(vm.ResourceGroup, vm.Name, nil)
	if err != nil {
		return err
	}

	// Make sure VM is deleted
	poller := newPoller(func() (string, error) {
		result, err := virtualMachinesClient.Get(vm.ResourceGroup, vm.Name, "InstanceView")
		if result.Properties != nil && result.Properties.InstanceView != nil {
			state := *(*result.Properties.InstanceView.Statuses)[1].DisplayStatus
			return translateState(state), err
		}

		return lvm.VMUnknown, err
	})

	_, err = poller.pollAsNeeded()
	if err != nil && !strings.Contains(err.Error(), `Code="ResourceNotFound"`) {
		return err
	}

	// Delete the OS File of this VM
	err = vm.deleteOSFile()
	if err != nil {
		return err
	}

	// Delete the network interface of this VM
	err = vm.deleteNic()
	if err != nil {
		return err
	}

	// Delete the public IP of this VM
	return vm.deletePublicIP()
}

// Halt shuts down the VM.
func (vm *VM) Halt() error {
	// Poweroff the VM
	virtualMachinesClient := compute.NewVirtualMachinesClient(vm.Creds.SubscriptionID)
	virtualMachinesClient.Authorizer = vm.Authorizer

	_, err := virtualMachinesClient.PowerOff(vm.ResourceGroup, vm.Name, nil)
	if err != nil {
		return err
	}

	// Make sure the VM is stopped
	poller := newPoller(func() (string, error) {
		result, err := virtualMachinesClient.Get(vm.ResourceGroup, vm.Name, "InstanceView")
		if result.Properties != nil && result.Properties.InstanceView != nil {
			state := *(*result.Properties.InstanceView.Statuses)[1].DisplayStatus
			return state, err
		}

		return lvm.VMUnknown, err
	})

	state, err := poller.pollAsNeeded()
	if err != nil {
		return err
	}

	if state != stopped {
		return fmt.Errorf("halt failed with a status of '%s'", state)
	}
	return nil
}

// Start boots a stopped VM.
func (vm *VM) Start() error {
	// Start the VM
	virtualMachinesClient := compute.NewVirtualMachinesClient(vm.Creds.SubscriptionID)
	virtualMachinesClient.Authorizer = vm.Authorizer

	_, err := virtualMachinesClient.Start(vm.ResourceGroup, vm.Name, nil)
	if err != nil {
		return err
	}

	// Make sure the VM is running
	poller := newPoller(func() (string, error) {
		result, err := virtualMachinesClient.Get(vm.ResourceGroup, vm.Name, "InstanceView")
		if result.Properties != nil && result.Properties.InstanceView != nil {
			state := *(*result.Properties.InstanceView.Statuses)[1].DisplayStatus
			return state, err
		}

		return lvm.VMUnknown, err
	})

	state, err := poller.pollAsNeeded()
	if err != nil {
		return err
	}

	if state != running {
		return fmt.Errorf("start failed with a status of '%s'", state)
	}
	return nil
}

// Suspend returns an error because it is not supported on Azure.
func (vm *VM) Suspend() error {
	return lvm.ErrSuspendNotSupported
}

// Resume returns an error because it is not supported on Azure.
func (vm *VM) Resume() error {
	return lvm.ErrResumeNotSupported
}
