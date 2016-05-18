// Copyright 2016 Apcera Inc. All rights reserved.

package arm

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"

	armStorage "github.com/Azure/azure-sdk-for-go/arm/storage"
	lvm "github.com/apcera/libretto/virtualmachine"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure"
)

// NewServicePrincipalTokenFromCredentials creates a new ServicePrincipalToken using values of the
// passed credentials map.
func NewServicePrincipalTokenFromCredentials(creds *OAuthCredentials, scope string) (*azure.ServicePrincipalToken, error) {
	oauthConfig, err := azure.PublicCloud.OAuthConfigForTenant(creds.TenantID)
	if err != nil {
		return nil, err
	}
	return azure.NewServicePrincipalToken(*oauthConfig, creds.ClientID, creds.ClientSecret, scope)
}

type templateParameter struct {
	Value string `json:"value"`
}

type templateParameters struct {
	AdminUsername        *templateParameter `json:"adminUsername,omitempty"`
	AdminPassword        *templateParameter `json:"adminPassword,omitempty"`
	ImageOffer           *templateParameter `json:"imageOffer,omitempty"`
	ImagePublisher       *templateParameter `json:"imagePublisher,omitempty"`
	ImageSku             *templateParameter `json:"imageSku,omitempty"`
	NetworkSecurityGroup *templateParameter `json:"networkSecurityGroup,omitempty"`
	NicName              *templateParameter `json:"nicName,omitempty"`
	OSFileName           *templateParameter `json:"osFileName,omitempty"`
	PublicIPName         *templateParameter `json:"publicIPName,omitempty"`
	SSHAuthorizedKey     *templateParameter `json:"sshAuthorizedKey,omitempty"`
	SubnetName           *templateParameter `json:"subnetName,omitempty"`
	VirtualNetworkName   *templateParameter `json:"virtualNetworkName,omitempty"`
	StorageAccountName   *templateParameter `json:"storageAccountName,omitempty"`
	StorageContainerName *templateParameter `json:"storageContainerName,omitempty"`
	VMSize               *templateParameter `json:"vmSize,omitempty"`
	VMName               *templateParameter `json:"vmName,omitempty"`
}

type poller struct {
	getProvisioningState func() (string, error)
	pause                func()
}

func newPoller(getProvisioningState func() (string, error)) *poller {
	pollDuration := time.Second * 1

	return &poller{
		getProvisioningState: getProvisioningState,
		pause:                func() { time.Sleep(pollDuration) },
	}
}

func (t *poller) pollAsNeeded() (string, error) {
	for {
		res, err := t.getProvisioningState()
		if err != nil {
			return res, err
		}

		switch res {
		case running, stopped, succeeded:
			return res, nil
		}

		t.pause()
	}
}

// Translates the given VM to arm template parameters
func (vm *VM) toTemplateParameters() *templateParameters {
	return &templateParameters{
		AdminUsername:        &templateParameter{vm.SSHCreds.SSHUser},
		AdminPassword:        &templateParameter{vm.SSHCreds.SSHPassword},
		ImageOffer:           &templateParameter{vm.ImageOffer},
		ImagePublisher:       &templateParameter{vm.ImagePublisher},
		ImageSku:             &templateParameter{vm.ImageSku},
		NetworkSecurityGroup: &templateParameter{vm.NetworkSecurityGroup},
		NicName:              &templateParameter{vm.Nic},
		OSFileName:           &templateParameter{vm.OsFile},
		PublicIPName:         &templateParameter{vm.PublicIP},
		SSHAuthorizedKey:     &templateParameter{vm.SSHCreds.SSHPrivateKey},
		StorageAccountName:   &templateParameter{vm.StorageAccount},
		StorageContainerName: &templateParameter{vm.StorageContainer},
		SubnetName:           &templateParameter{vm.Subnet},
		VirtualNetworkName:   &templateParameter{vm.VirtualNetwork},
		VMSize:               &templateParameter{vm.Size},
		VMName:               &templateParameter{vm.Name},
	}
}

// validateVM validates the members of given VM object
func validateVM(vm *VM) error {
	// Validate the OAUTH Credentials
	if vm.Creds.ClientID == "" {
		return fmt.Errorf("a client id must be specified")
	}

	if vm.Creds.ClientSecret == "" {
		return fmt.Errorf("a client secret must be specified")
	}

	if vm.Creds.TenantID == "" {
		return fmt.Errorf("a tenant id must be specified")
	}

	if vm.Creds.SubscriptionID == "" {
		return fmt.Errorf("a subscription id must be specified")
	}

	// Validate the image
	if vm.ImagePublisher == "" {
		return fmt.Errorf("an image publisher must be specified")
	}

	if vm.ImageOffer == "" {
		return fmt.Errorf("an image offer must be specified")
	}

	if vm.ImageSku == "" {
		return fmt.Errorf("an image sku must be specified")
	}

	// Validate the deployment
	if vm.ResourceGroup == "" {
		return fmt.Errorf("a resource group must be specified")
	}

	if vm.StorageAccount == "" {
		return fmt.Errorf("a storage account must be specified")
	}

	// Validate the network
	if vm.NetworkSecurityGroup == "" {
		return fmt.Errorf("a network security group must be specified")
	}

	if vm.Subnet == "" {
		return fmt.Errorf("a subnet must be specified")
	}

	if vm.VirtualNetwork == "" {
		return fmt.Errorf("a virtual network must be specified")
	}
	return nil
}

// deploy deploys the given VM based on the default Linux arm template over the
// VM's resource group.
func (vm *VM) deploy() error {
	// Pass the parameters to the arm templacte
	var templateParameters = vm.toTemplateParameters()
	factory := deploymentFactory{template: Linux}
	deployment, err := factory.create(*templateParameters)
	if err != nil {
		return err
	}

	// Create and send the deployment to the resource group
	deploymentsClient := resources.NewDeploymentsClient(vm.Creds.SubscriptionID)
	deploymentsClient.Authorizer = vm.Authorizer

	_, err = deploymentsClient.CreateOrUpdate(vm.ResourceGroup, deploymentName, *deployment, nil)
	if err != nil {
		return err
	}

	// Make sure the deployment is succeeded
	poller := newPoller(func() (string, error) {
		result, err := deploymentsClient.Get(vm.ResourceGroup, deploymentName)
		if result.Properties != nil && result.Properties.ProvisioningState != nil {
			return *result.Properties.ProvisioningState, err
		}

		return lvm.VMUnknown, err
	})

	pollStatus, err := poller.pollAsNeeded()
	if err != nil {
		return err
	}

	if pollStatus != succeeded {
		return fmt.Errorf("deployment failed with a status of '%s'", pollStatus)
	}
	return nil
}

// getPublicIP returns the public IP of the given VM, if exists one.
func (vm *VM) getPublicIP() (net.IP, error) {
	publicIPAddressesClient := network.NewPublicIPAddressesClient(vm.Creds.SubscriptionID)
	publicIPAddressesClient.Authorizer = vm.Authorizer

	resPublicIP, err := publicIPAddressesClient.Get(vm.ResourceGroup, vm.PublicIP, "")
	if err != nil {
		return nil, err
	}

	if resPublicIP.Properties == nil || *resPublicIP.Properties.IPAddress == "" {
		return nil, fmt.Errorf("VM has no public IP address")
	}
	return net.ParseIP(*resPublicIP.Properties.IPAddress), nil
}

// getPrivateIP returns the private IP of the given VM, if exists one.
func (vm *VM) getPrivateIP() (net.IP, error) {
	interfaceClient := network.NewInterfacesClient(vm.Creds.SubscriptionID)
	interfaceClient.Authorizer = vm.Authorizer

	resPrivateIP, err := interfaceClient.Get(vm.ResourceGroup, vm.Nic, "")
	if err != nil {
		return nil, err
	}

	if resPrivateIP.Properties == nil || len(*resPrivateIP.Properties.IPConfigurations) == 0 {
		return nil, fmt.Errorf("VM has no private IP address")
	}
	ipConfigs := *resPrivateIP.Properties.IPConfigurations
	if len(ipConfigs) > 1 {
		return nil, fmt.Errorf("VM has multiple private IP addresses")
	}
	return net.ParseIP(*ipConfigs[0].Properties.PrivateIPAddress), nil
}

// deleteOSFile deletes the OS file from the VM's storage account, returns an error if the operation
// does not succeed.
func (vm *VM) deleteOSFile() error {
	storageAccountsClient := armStorage.NewAccountsClient(vm.Creds.SubscriptionID)
	storageAccountsClient.Authorizer = vm.Authorizer

	accountKeys, err := storageAccountsClient.ListKeys(vm.ResourceGroup, vm.StorageAccount)
	if err != nil {
		return err
	}

	storageClient, err := storage.NewBasicClient(vm.StorageAccount, *accountKeys.Key1)
	if err != nil {
		return err
	}

	blobStorageClient := storageClient.GetBlobService()
	err = blobStorageClient.DeleteBlob(vm.StorageContainer, vm.OsFile, nil)
	return err
}

// deleteNic deletes the network interface for the given VM from the VM's resource group, returns an error
// if the operation does not succeed.
func (vm *VM) deleteNic() error {
	interfaceClient := network.NewInterfacesClient(vm.Creds.SubscriptionID)
	interfaceClient.Authorizer = vm.Authorizer

	_, err := interfaceClient.Delete(vm.ResourceGroup, vm.Nic, nil)
	return err
}

// deletePublicIP deletes the reserved Public IP of the given VM from the VM's resource group, returns an error
// if the operation does not succeed.
func (vm *VM) deletePublicIP() error {
	// Delete the Public IP of this VM
	publicIPAddressesClient := network.NewPublicIPAddressesClient(vm.Creds.SubscriptionID)
	publicIPAddressesClient.Authorizer = vm.Authorizer

	_, err := publicIPAddressesClient.Delete(vm.ResourceGroup, vm.PublicIP, nil)
	return err
}

type deploymentFactory struct {
	template string
}

func (f *deploymentFactory) create(tempParams templateParameters) (*resources.Deployment, error) {
	template, err := f.getTemplate()
	if err != nil {
		return nil, err
	}

	parameters, err := f.getTemplateParameters(tempParams)
	if err != nil {
		return nil, err
	}

	return &resources.Deployment{
		Properties: &resources.DeploymentProperties{
			Mode:       resources.Incremental,
			Template:   template,
			Parameters: parameters,
		},
	}, nil
}

func (f *deploymentFactory) getTemplate() (*map[string]interface{}, error) {
	var t map[string]interface{}
	err := json.Unmarshal([]byte(f.template), &t)

	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (f *deploymentFactory) getTemplateParameters(tempParams templateParameters) (*map[string]interface{}, error) {
	b, err := json.Marshal(tempParams)
	if err != nil {
		return nil, err
	}

	var t map[string]interface{}
	err = json.Unmarshal(b, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// translateState converts an Azure state to a libretto state.
func translateState(azureState string) string {
	switch azureState {
	case running:
		return lvm.VMRunning
	case stopped:
		return lvm.VMHalted
	default:
		return lvm.VMUnknown
	}
}

func randStringRunes(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[r.Intn(len(letterRunes))]
	}
	return string(b)
}
