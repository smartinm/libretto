// Copyright 2015 Apcera Inc. All rights reserved.

package openstack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud"
	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud/openstack"
	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud/openstack/blockstorage/v1/volumes"
	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud/openstack/compute/v2/extensions/volumeattach"
	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud/openstack/compute/v2/images"
	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud/openstack/compute/v2/servers"

	"github.com/apcera/libretto/ssh"
	lvm "github.com/apcera/libretto/virtualmachine"
)

func getProviderClient(vm *VM) (*gophercloud.ProviderClient, error) {
	// Set the opts to autheticate clients. For now, we only support basic auth (host, username, password)
	// Or user can download its Openstack RC File and source it to its console, then opts will be read via ENV_VARS
	var opts gophercloud.AuthOptions
	var err error
	if vm.Username == "" || vm.Password == "" {
		opts, err = openstack.AuthOptionsFromEnv()

		if err != nil {
			return nil, ErrAuthOptions
		}
	} else {
		opts = gophercloud.AuthOptions{
			IdentityEndpoint: vm.IdentityEndpoint,
			Username:         vm.Username,
			Password:         vm.Password,
			TenantName:       vm.TenantName,
		}
	}

	providerClient, err := openstack.AuthenticatedClient(opts)
	if providerClient == nil || err != nil {
		return nil, fmt.Errorf("Failed to authenticate the client")
	}

	return providerClient, nil
}

func getComputeClient(vm *VM) (*gophercloud.ServiceClient, error) {
	if vm.computeClient != nil {
		return vm.computeClient, nil
	}

	provider, err := getProviderClient(vm)
	if err != nil {
		return nil, ErrAuthenticatingClient
	}

	endpointOpts := gophercloud.EndpointOpts{
		Region: vm.Region,
	}

	client, err := openstack.NewComputeV2(provider, endpointOpts)
	if err != nil {
		return nil, ErrInvalidRegion
	}

	vm.computeClient = client
	return client, nil
}

func getNetworkClient(vm *VM) (*gophercloud.ServiceClient, error) {
	provider, err := getProviderClient(vm)
	if err != nil {
		return nil, ErrAuthenticatingClient
	}

	endpointOpts := gophercloud.EndpointOpts{
		Region: vm.Region,
	}

	client, err := openstack.NewNetworkV2(provider, endpointOpts)
	if err != nil {
		return nil, ErrInvalidRegion
	}
	return client, nil
}

func getBlockStorageClient(vm *VM) (*gophercloud.ServiceClient, error) {
	provider, err := getProviderClient(vm)
	if err != nil {
		return nil, ErrAuthenticatingClient
	}

	endpointOpts := gophercloud.EndpointOpts{
		Region: vm.Region,
	}

	client, err := openstack.NewBlockStorageV1(provider, endpointOpts)
	if err != nil {
		return nil, ErrInvalidRegion
	}
	return client, nil
}

// Reserves an Image ID at the specified image endpoint using the information in given imageMetadata
// Returns the reserved Image ID if reservation is successful, otherwise returns an error.
// Requires client's token to reserve the image.
func reserveImage(tokenID string, imageEndpoint string, imageMetadata ImageMetadata) (string, error) {
	imageStr, err := json.Marshal(imageMetadata)

	if err != nil {
		return "", err
	}

	imagesURI := fmt.Sprintf("%sv2/images", imageEndpoint)
	createReq, err := http.NewRequest("POST", imagesURI, bytes.NewBuffer(imageStr))

	createReq.Header.Add("Content-Type", "application/json")
	createReq.Header.Add("X-Auth-Token", tokenID)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(createReq)

	if err != nil {
		return "", fmt.Errorf("Failed to send a image reserve request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return "", fmt.Errorf("Reserve Image request returned bad response, %s", resp.Body)
	}

	body, _ := ioutil.ReadAll(resp.Body)

	var dat map[string]interface{}
	if err := json.Unmarshal(body, &dat); err != nil {
		return "", err
	}

	if dat["status"] != imageQueued {
		return "", fmt.Errorf("Image has never been queued")
	}

	// Retrieve the image ID from http response block
	idFromResponse := dat["id"]
	switch idFromResponse.(type) {
	case string:
		return idFromResponse.(string), nil
	default:
		return "", fmt.Errorf("Unable to parse the upload image response")
	}
}

// Uploads the image to an reserved image location at the imageEndpoint using the reserved image ID and imageMetadata.
// Returns nil error if the upload is successful, otherwise returns an error.
// Requires client's token to upload the image.
func uploadImage(tokenID string, imageEndpoint string, imageID string, imagePath string) error {
	file, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("Unable to open image file")
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("Unable to get the stats of the image file: %s", err)
	}
	imageFileSize := stat.Size()

	// Prepare the request to upload the image file
	imageLocation := fmt.Sprintf("%sv2/images/%s/file", imageEndpoint, imageID)
	uploadReq, err := http.NewRequest("PUT", imageLocation, file)
	if err != nil {
		return fmt.Errorf("Unable to upload image to the openstack")
	}

	uploadReq.Header.Add("Content-Type", "application/octet-stream")
	uploadReq.Header.Add("X-Auth-Token", tokenID)
	uploadReq.Header.Add("Content-Length", fmt.Sprintf("%d", imageFileSize))

	uploadClient := &http.Client{}

	// Send the request to upload the image
	resp, err := uploadClient.Do(uploadReq)
	if err != nil {
		return fmt.Errorf("Failed to send a upload image request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("Upload Image request returned bad response, %s", resp.Body)
	}

	return nil
}

// Creates an Image based on the given FilePath and returns the UUID of the image
func createImage(vm *VM) (string, error) {
	// Get the openstack provider
	provider, err := getProviderClient(vm)
	if err != nil {
		return "", ErrAuthenticatingClient
	}

	endpointOpts := gophercloud.EndpointOpts{
		Region: vm.Region,
	}
	// Find the Image Endpoint to upload the image
	imageEndpoint, err := findImageEndpoint(provider, endpointOpts)
	if err != nil {
		return "", err
	}

	// Reserve an ImageID at imageEndpoint using the given image metadata
	imageID, err := reserveImage(provider.TokenID, imageEndpoint, vm.ImageMetadata)
	if err != nil {
		return "", err
	}

	// Upload the image to the imageEndpoint with reserved ImageID using the given image path
	err = uploadImage(provider.TokenID, imageEndpoint, imageID, vm.ImagePath)
	if err != nil {
		return "", err
	}

	return imageID, nil
}

// getServer returns the Openstack server object for the VM. An error is returned
// if the instance ID is missing, if there was a problem querying Openstack, or if
// there is no instances with the given VM ID.
func getServer(vm *VM) (*servers.Server, error) {
	if vm.InstanceID == "" {
		// Probably need to call Provision first.
		return nil, ErrNoInstanceID
	}

	client, err := getComputeClient(vm)
	if err != nil {
		return nil, err
	}

	status, err := servers.Get(client, vm.InstanceID).Extract()
	if status != nil && err != nil {
		return nil, fmt.Errorf("Failed to retrieve the server for VM")
	}

	return status, nil
}

// Finds the image endpoint in the given openstack Region. Region is passed within gophercloud.EndpointOpts
func findImageEndpoint(client *gophercloud.ProviderClient, eo gophercloud.EndpointOpts) (string, error) {
	eo.ApplyDefaults("image")
	url, err := client.EndpointLocator(eo)
	if err != nil {
		return "", fmt.Errorf("Error on locating image endpoint")
	}
	return url, nil
}

// Waits until the given VM becomes in requested state in given ActionTimeout seconds
func waitUntil(vm *VM, state string) error {
	var curState string
	var err error
	for i := 0; i < ActionTimeout; i++ {
		curState, err = vm.GetState()
		if err != nil {
			return err
		}

		if curState == state {
			break
		}

		if curState == lvm.VMError {
			return fmt.Errorf("Failed to bring the VM to state: %s", state)
		}

		time.Sleep(1 * time.Second)
	}
	if curState != state {
		return ErrActionTimeout
	}
	return nil
}

// Waits until the given VM becomes ready. Basically, waits until vm can be sshed.
func waitUntilSSHReady(vm *VM) error {
	client, err := vm.GetSSH(ssh.Options{})
	if err != nil {
		return err
	}
	return client.WaitForSSH(SSHTimeout * time.Second)
}

// createAndAttachVolume creates a new volume with the given volume specs and then attaches this volume to the given VM.
func createAndAttachVolume(vm *VM) error {
	if vm.InstanceID == "" {
		// Probably need to call Provision first.
		return ErrNoInstanceID
	}

	cClient, err := getComputeClient(vm)
	if err != nil {
		return fmt.Errorf("Compute Client is not set for the VM, %s", err)
	}

	bsClient, err := getBlockStorageClient(vm)
	if err != nil {
		return err
	}

	// Creates a new Volume for this VM
	volume := vm.Volume
	vOpts := volumes.CreateOpts{Size: volume.Size, Name: volume.Name, VolumeType: volume.Type}
	vol, err := volumes.Create(bsClient, vOpts).Extract()
	if err != nil {
		return fmt.Errorf("Failed to create a new volume for the VM: %s", err)
	}

	// Wait until Volume becomes available
	err = waitUntilVolume(bsClient, vol.ID, volumeStateAvailable)
	if err != nil {
		return fmt.Errorf("Failed to create a new volume for the VM: %s", err)
	}

	// Attach the new volume to this VM
	vaOpts := volumeattach.CreateOpts{Device: volume.Device, VolumeID: vol.ID}
	va, err := volumeattach.Create(cClient, vm.InstanceID, vaOpts).Extract()
	if err != nil {
		return fmt.Errorf("Failed to attach the volume to the VM: %s", err)
	}

	// Wait until Volume is attached to the VM
	err = waitUntilVolume(bsClient, vol.ID, volumeStateInUse)
	if err != nil {
		return fmt.Errorf("Failed to attach the volume to the VM: %s", err)
	}

	vm.Volume.ID = vol.ID
	vm.Volume.Device = va.Device

	return nil
}

// deattachAndDeleteVolume deattaches the volume from the given VM and then completely deletes the volume.
func deattachAndDeleteVolume(vm *VM) error {
	if vm.InstanceID == "" {
		// Probably need to call Provision first.
		return ErrNoInstanceID
	}

	cClient, err := getComputeClient(vm)
	if err != nil {
		return fmt.Errorf("Compute Client is not set for the VM, %s", err)
	}

	bsClient, err := getBlockStorageClient(vm)
	if err != nil {
		return err
	}

	// Deattach the volume from the VM
	err = volumeattach.Delete(cClient, vm.InstanceID, vm.Volume.ID).ExtractErr()
	if err != nil {
		return fmt.Errorf("Failed to deattach volume from the VM: %s", err)
	}

	// Wait until Volume is de-attached from the VM
	err = waitUntilVolume(bsClient, vm.Volume.ID, volumeStateAvailable)
	if err != nil {
		return fmt.Errorf("Failed to deattach volume from the VM: %s", err)
	}

	// Delete the volume
	err = volumes.Delete(bsClient, vm.Volume.ID).ExtractErr()
	if err != nil {
		return fmt.Errorf("Failed to delete volume: %s", err)
	}

	// Wait until Volume is deleted
	err = waitUntilVolume(bsClient, vm.Volume.ID, volumeStateDeleted)
	if err != nil {
		return fmt.Errorf("Failed to delete volume: %s", err)
	}

	return nil
}

// findImageIDByName finds the ImageID for the given imageName, returns an error if there is
// no image or more than one image with the given Image Name.
func findImageIDByName(client *gophercloud.ServiceClient, imageName string) (string, error) {
	if imageName == "" {
		return "", fmt.Errorf("Empty image name")
	}

	// We have the option of filtering the image list. If we want the full
	// collection, leave it as an empty struct
	opts := images.ListOpts{Name: imageName}

	// Retrieve image list
	page, err := images.ListDetail(client, opts).AllPages()
	if err != nil {
		return "", fmt.Errorf("Error on retrieving image pages: %s", err)
	}

	imageList, err := images.ExtractImages(page)
	if err != nil {
		return "", fmt.Errorf("Error on extracting image list: %s", err)
	}

	if len(imageList) == 0 {
		return "", nil
	}

	if len(imageList) > 1 {
		return "", fmt.Errorf("There exists more than one image with the same name")
	}

	return imageList[0].ID, err
}

// waitUntilVolume waits until the given volume turns into given state under given VolumeActionTimeout seconds
func waitUntilVolume(blockStorateClient *gophercloud.ServiceClient, volumeID string, state string) error {
	for i := 0; i < VolumeActionTimeout; i++ {
		vol, err := volumes.Get(blockStorateClient, volumeID).Extract()
		switch {
		case vol == nil && state == "nil":
			return nil
		case vol == nil || err != nil:
			return fmt.Errorf("Failed on getting volume Status: %s", err)
		case vol.Status == state:
			return nil
		case vol.Status == lvm.VMError || vol.Status == volumeStateErrorDeleting:
			return fmt.Errorf("Failed to bring the volume to state %s, ended up at state %s", state, vol.Status)
		}
		time.Sleep(1 * time.Second)
	}
	return ErrActionTimeout
}

// NewDefaultImageMetadata creates a ImageMetadata with default values
func NewDefaultImageMetadata() ImageMetadata {
	return ImageMetadata{
		ContainerFormat: "bare",
		DiskFormat:      "qcow2",
		MinDisk:         10,
		MinRAM:          1024,
		Name:            "new-image",
	}
}

// NewDefaultVolume creates a Volume with default values
func NewDefaultVolume() Volume {
	return Volume{
		Name:   "test",
		Size:   10,
		Device: "/dev/vdb",
	}
}
