// Copyright 2016 Apcera Inc. All rights reserved.
package google

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"

	"github.com/apcera/util/uuid"

	gce "google.golang.org/api/compute/v1"
)

var tokenURL = "https://accounts.google.com/o/oauth2/token"

type GCEService struct {
	vm      *VM
	service *gce.Service
}

// credFile represents the structure of the account file JSON file.
type credFile struct {
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	ClientId    string `json:"client_id"`
}

// Initialize default parameters
func (vm *VM) init() {
	if vm.Name == "" {
		vm.Name = fmt.Sprintf("libretto-vm-%s", uuid.Variant4())
	}

	if vm.Zone == "" {
		vm.Zone = defaultZone
	}

	if vm.SourceImage == "" {
		vm.SourceImage = defaultSourceImage
	}

	if vm.MachineType == "" {
		vm.MachineType = defaultMachineType
	}

	if vm.DiskType == "" {
		vm.DiskType = defaultDiskType
	}

	if vm.DiskSize <= 0 {
		vm.DiskSize = defaultDiskSize
	}

	if vm.Scopes == nil {
		vm.Scopes = defaultScopes
	}
}

func (vm *VM) getService() (*GCEService, error) {
	var err error
	var client *http.Client

	s := &GCEService{}

	if err := parseAccountFile(&vm.account, vm.AccountFile); err != nil {
		return s, err
	}

	// Auth with AccountFile first if provided
	if vm.account.PrivateKey != "" {
		config := jwt.Config{
			Email:      vm.account.ClientEmail,
			PrivateKey: []byte(vm.account.PrivateKey),
			Scopes:     vm.Scopes,
			TokenURL:   tokenURL,
		}

		client = config.Client(oauth2.NoContext)
	} else {
		client = &http.Client{
			Transport: &oauth2.Transport{
				Source: google.ComputeTokenSource(""),
			},
		}
	}

	svc, err := gce.New(client)
	if err != nil {
		return s, err
	}

	vm.init()

	s = &GCEService{
		vm:      vm,
		service: svc,
	}

	return s, nil
}

// get instance from current VM definition
func (svc *GCEService) getInstance() (*gce.Instance, error) {
	return svc.service.Instances.Get(svc.vm.Project, svc.vm.Zone, svc.vm.Name).Do()
}

// pullOperationStatus pulls to wait for the operation to finish.
func pullOperationStatus(funcOperation func() (*gce.Operation, error)) error {
	for {
		op, err := funcOperation()
		if err != nil {
			return err
		}

		fmt.Println(fmt.Sprintf("operation %q status: %s", op.Name, op.Status))
		if op.Status == "DONE" {
			if op.Error != nil {
				return fmt.Errorf("operation error: %v", *op.Error.Errors[0])
			}
			break
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

// waitForOperationReady waits for the regional operation to finish.
func (svc *GCEService) waitForOperationReady(operation string) error {
	return pullOperationStatus(func() (*gce.Operation, error) {
		return svc.service.ZoneOperations.Get(svc.vm.Project, svc.vm.Zone, operation).Do()
	})
}

func (svc *GCEService) getImage() (image *gce.Image, err error) {
	imgFamily := []string{
		"centos-cloud",
		"coreos-cloud",
		"debian-cloud",
		"google-containers",
		"opensuse-cloud",
		"rhel-cloud",
		"suse-cloud",
		"ubuntu-os-cloud",
	}

	for _, img := range imgFamily {
		image, err := svc.service.Images.Get(img, svc.vm.SourceImage).Do()
		if err == nil && image != nil && image.SelfLink != "" {
			return image, nil
		}
		image = nil
	}

	err = fmt.Errorf("Image %s could not be found in any of these projects: %s", svc.vm.SourceImage, imgFamily)
	return nil, err
}

// Returns the IP addresses of the GCE instance.
func (svc *GCEService) getIPs() ([]net.IP, error) {
	instance, err := svc.service.Instances.Get(svc.vm.Project, svc.vm.Zone, svc.vm.Name).Do()
	if err != nil {
		return nil, err
	}

	ips := make([]net.IP, 2)
	nic := instance.NetworkInterfaces[0]

	publicIP := nic.AccessConfigs[0].NatIP
	if publicIP == "" {
		return nil, errors.New("error to retrieve public IP")
	}

	privateIP := nic.NetworkIP
	if privateIP == "" {
		return nil, errors.New("error to retrieve private IP")
	}

	ips[PublicIP] = net.ParseIP(publicIP)
	ips[PrivateIP] = net.ParseIP(privateIP)

	return ips, nil
}

// provision a new GCE VM instance
func (svc *GCEService) provision() error {
	// Get zone
	zone, err := svc.service.Zones.Get(svc.vm.Project, svc.vm.Zone).Do()
	if err != nil {
		return err
	}

	// Get source image
	image, err := svc.getImage()
	if err != nil {
		return err
	}

	// Get GCE machine type
	machineType, err := svc.service.MachineTypes.Get(svc.vm.Project, zone.Name, svc.vm.MachineType).Do()
	if err != nil {
		return err
	}

	// Get Network
	network, err := svc.service.Networks.Get(svc.vm.Project, svc.vm.Network).Do()
	if err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf("get network %v", network))

	// Subnetwork
	// Validate Subnetwork config now that we have some info about the network
	if !network.AutoCreateSubnetworks && len(network.Subnetworks) > 0 {
		// Network appears to be in "custom" mode, so a subnetwork is required
		if svc.vm.Subnetwork == "" {
			return fmt.Errorf("a subnetwork must be specified")
		}
	}
	// Get the subnetwork
	subnetworkSelfLink := ""
	if svc.vm.Subnetwork != "" {
		subnetwork, err := svc.service.Subnetworks.Get(svc.vm.Project, svc.vm.region(), svc.vm.Subnetwork).Do()
		if err != nil {
			return err
		}
		subnetworkSelfLink = subnetwork.SelfLink
	}

	// If given a regional ip, get it
	accessconfig := gce.AccessConfig{
		Name: "External NAT for Libretto",
		Type: "ONE_TO_ONE_NAT",
	}

	metaData := fmt.Sprintf("%s:%s\n", svc.vm.SSHCreds.SSHUser, svc.vm.SSHPublicKey)

	// Create the instance information
	instance := &gce.Instance{
		Name:        svc.vm.Name,
		Description: "libretto vm",
		Disks: []*gce.AttachedDisk{
			&gce.AttachedDisk{
				Type:       "PERSISTENT",
				Mode:       "READ_WRITE",
				Kind:       "compute#attachedDisk",
				Boot:       true,
				AutoDelete: true,
				InitializeParams: &gce.AttachedDiskInitializeParams{
					SourceImage: image.SelfLink,
					DiskSizeGb:  int64(svc.vm.DiskSize),
					DiskType:    fmt.Sprintf("zones/%s/diskTypes/%s", zone.Name, svc.vm.DiskType),
				},
			},
		},
		MachineType: machineType.SelfLink,
		Metadata: &gce.Metadata{
			Items: []*gce.MetadataItems{
				{
					Key:   "sshKeys",
					Value: &metaData,
				},
			},
		},
		NetworkInterfaces: []*gce.NetworkInterface{
			&gce.NetworkInterface{
				AccessConfigs: []*gce.AccessConfig{
					&accessconfig,
				},
				Network:    network.SelfLink,
				Subnetwork: subnetworkSelfLink,
			},
		},
		Scheduling: &gce.Scheduling{
			Preemptible: svc.vm.Preemptible,
		},
		ServiceAccounts: []*gce.ServiceAccount{
			&gce.ServiceAccount{
				Email:  "default",
				Scopes: svc.vm.Scopes,
			},
		},
		Tags: &gce.Tags{
			Items: svc.vm.Tags,
		},
	}

	op, err := svc.service.Instances.Insert(svc.vm.Project, zone.Name, instance).Do()
	if err != nil {
		return err
	}

	if err = svc.waitForOperationReady(op.Name); err != nil {
		return err
	}

	_, err = svc.getInstance()
	return err
}

// starts an stopped GCE instance.
func (svc *GCEService) start() error {

	instance, err := svc.getInstance()
	if err != nil {
		if !strings.Contains(err.Error(), "no instance found") {
			return err
		}
	}

	if instance == nil {
		return errors.New("no instance found")
	}

	op, err := svc.service.Instances.Start(svc.vm.Project, svc.vm.Zone, svc.vm.Name).Do()
	if err != nil {
		return err
	}

	fmt.Println("Waiting for instance to start")
	return svc.waitForOperationReady(op.Name)
}

// starts an stopped GCE instance.
func (svc *GCEService) stop() error {

	instance, err := svc.getInstance()
	if err != nil {
		if !strings.Contains(err.Error(), "no instance found") {
			return err
		}
	}

	if instance == nil {
		return errors.New("no instance found")
	}

	op, err := svc.service.Instances.Stop(svc.vm.Project, svc.vm.Zone, svc.vm.Name).Do()
	if err != nil {
		return err
	}

	fmt.Println("Waiting for instance to stop")
	return svc.waitForOperationReady(op.Name)
}

// deletes the GCE instance.
func (svc *GCEService) delete() error {
	op, err := svc.service.Instances.Delete(svc.vm.Project, svc.vm.Zone, svc.vm.Name).Do()
	if err != nil {
		return err
	}

	fmt.Println("Waiting for instance to be deleted.")
	return svc.waitForOperationReady(op.Name)
}

// extract the region from zone name
func (vm *VM) region() string {
	return vm.Zone[:len(vm.Zone)-2]
}

func parseAccountJSON(result interface{}, jsonText string) error {
	dec := json.NewDecoder(strings.NewReader(jsonText))
	return dec.Decode(result)
}

func parseAccountFile(file *credFile, jsonText string) error {
	err := parseAccountJSON(file, jsonText)
	if err != nil {
		if _, err = os.Stat(jsonText); os.IsNotExist(err) {
			return fmt.Errorf("error finding account file: %s", jsonText)
		}

		bytes, err := ioutil.ReadFile(jsonText)
		if err != nil {
			return fmt.Errorf("error reading account file from path '%s': %s", jsonText, err)
		}

		if err := parseAccountJSON(file, string(bytes)); err != nil {
			return fmt.Errorf("error parsing account file: %s", err)
		}
	}

	return nil
}
