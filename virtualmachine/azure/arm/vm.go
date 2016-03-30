// Copyright 2016 Apcera Inc. All rights reserved.

// Package azure provides methods for creating and manipulating VMs on Azure.
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
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

const (
	// PublicIP is the index of the public IP address that GetIPs returns.
	PublicIP = 0

	// PrivateIP is the index of the private IP address that GetIPs returns.
	PrivateIP = 1

	// DefaultTimeout is the maximum seconds to wait before failing to GetSSH.
	DefaultTimeout = 800

	// VM Statuses
	Running = "VM running"
	Stopped = "VM stopped"

	// Deployment Statuses
	Succeeded = "Succeeded"

	// Deployment name when provisioning a VM
	deploymentName = "libretto"
)

var _ lvm.VirtualMachine = (*VM)(nil)

// Authentication via OAUTH
type OAuthCredentials struct {
	ClientID       string `mapstructure:"client_id"`
	ClientSecret   string `mapstructure:"client_secret"`
	TenantID       string `mapstructure:"tenant_id"`
	SubscriptionID string `mapstructure:"subscription_id"`
}

// VM represents an Azure virtual machine.
type VM struct {
	// Credentials to connect Azure
	Creds OAuthCredentials

	// Image Properties
	ImagePublisher string `mapstructure:"image_publisher"`
	ImageOffer     string `mapstructure:"image_offer"`
	ImageSku       string `mapstructure:"image_sku"`
	Location       string `mapstructure:"location"`

	// VM Properties
	Size string `mapstructure:"vm_size"`
	Name string `mapstructure:"vm_name"`

	// SSH Properties
	SSHCreds ssh.Credentials // required

	// Deployment Properties
	ResourceGroup  string `mapstructure:"resource_group"`
	StorageAccount string `mapstructure:"storage_account"`

	// Authorizer to connect to Azure
	authorizer autorest.Authorizer

	// VM OS Properties
	osFile string `mapstructure:"os_file"`

	// VM Network Properties
	publicIP       string `mapstructure:"public_ip"`
	nic            string `mapstructure:"nic"`
	Subnet         string `mapstructure:"subnet"`
	VirtualNetwork string `mapstructure:"virtual_network"`
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
	vm.authorizer = spt

	// Set up private members of the VM
	tempName := fmt.Sprintf("%s-%s", vm.Name, randStringRunes(6))
	vm.osFile = tempName + "-os-disk.vhd"
	vm.publicIP = tempName + "-public-ip"
	vm.nic = tempName + "-nic"

	// Create and send the deployment
	vm.deploy()

	// Use GetSSH to try to connect to machine
	cli, err := vm.GetSSH(ssh.Options{KeepAlive: 2})
	if err != nil {
		return err
	}

	err = cli.WaitForSSH(DefaultTimeout * time.Second)
	if err != nil {
		return err
	}

	return nil
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
//     "Running"
//     "Halted"
func (vm *VM) GetState() (string, error) {
	virtualMachinesClient := compute.NewVirtualMachinesClient(vm.Creds.SubscriptionID)
	virtualMachinesClient.Authorizer = vm.authorizer

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
	virtualMachinesClient.Authorizer = vm.authorizer

	_, err := virtualMachinesClient.Delete(vm.ResourceGroup, vm.Name, nil)
	if err != nil {
		return err
	}

	// Make sure VM is deleted
	poller := NewPoller(func() (string, error) {
		r, e := virtualMachinesClient.Get(vm.ResourceGroup, vm.Name, "InstanceView")
		if r.Properties != nil && r.Properties.InstanceView != nil {
			state := *(*r.Properties.InstanceView.Statuses)[1].DisplayStatus
			return translateState(state), e
		}

		return "UNKNOWN", e
	})

	_, err = poller.PollAsNeeded()
	if err != nil && !strings.Contains(err.Error(), "Code=\"ResourceNotFound\"") {
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
	err = vm.deletePublicIP()
	if err != nil {
		return err
	}
	return nil
}

// Halt shuts down the VM.
func (vm *VM) Halt() error {
	// Poweroff the VM
	virtualMachinesClient := compute.NewVirtualMachinesClient(vm.Creds.SubscriptionID)
	virtualMachinesClient.Authorizer = vm.authorizer

	_, err := virtualMachinesClient.PowerOff(vm.ResourceGroup, vm.Name, nil)
	if err != nil {
		return err
	}

	// Make sure the VM is stopped
	poller := NewPoller(func() (string, error) {
		r, e := virtualMachinesClient.Get(vm.ResourceGroup, vm.Name, "InstanceView")
		if r.Properties != nil && r.Properties.InstanceView != nil {
			state := *(*r.Properties.InstanceView.Statuses)[1].DisplayStatus
			return state, e
		}

		return "UNKNOWN", e
	})

	state, err := poller.PollAsNeeded()
	if err != nil {
		return err
	}

	if state != Stopped {
		return fmt.Errorf("halt failed with a status of '%s'.", state)
	}
	return nil
}

// Start boots a stopped VM.
func (vm *VM) Start() error {
	// Start the VM
	virtualMachinesClient := compute.NewVirtualMachinesClient(vm.Creds.SubscriptionID)
	virtualMachinesClient.Authorizer = vm.authorizer

	_, err := virtualMachinesClient.Start(vm.ResourceGroup, vm.Name, nil)
	if err != nil {
		return err
	}

	// Make sure the VM is running
	poller := NewPoller(func() (string, error) {
		r, e := virtualMachinesClient.Get(vm.ResourceGroup, vm.Name, "InstanceView")
		if r.Properties != nil && r.Properties.InstanceView != nil {
			state := *(*r.Properties.InstanceView.Statuses)[1].DisplayStatus
			return state, e
		}

		return "UNKNOWN", e
	})

	state, err := poller.PollAsNeeded()
	if err != nil {
		return err
	}

	if state != Running {
		return fmt.Errorf("start failed with a status of '%s'.", state)
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
