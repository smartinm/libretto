// Copyright 2016 Apcera Inc. All rights reserved.

package arm

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"time"

	lvm "github.com/apcera/libretto/virtualmachine"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	armStorage "github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

// ToJSON returns the passed item as a pretty-printed JSON string. If any JSON error occurs,
// it returns the empty string.
func ToJSON(v interface{}) string {
	j, _ := json.MarshalIndent(v, "", "  ")
	return string(j)
}

// NewServicePrincipalTokenFromCredentials creates a new ServicePrincipalToken using values of the
// passed credentials map.
func NewServicePrincipalTokenFromCredentials(creds *OAuthCredentials, scope string) (*azure.ServicePrincipalToken, error) {
	oauthConfig, err := azure.PublicCloud.OAuthConfigForTenant(creds.TenantID)
	if err != nil {
		panic(err)
	}
	return azure.NewServicePrincipalToken(*oauthConfig, creds.ClientID, creds.ClientSecret, scope)
}

type TemplateParameter struct {
	Value string `json:"value"`
}

type TemplateParameters struct {
	AdminUsername      *TemplateParameter `json:"adminUsername,omitempty"`
	AdminPassword      *TemplateParameter `json:"adminPassword,omitempty"`
	ImageOffer         *TemplateParameter `json:"imageOffer,omitempty"`
	ImagePublisher     *TemplateParameter `json:"imagePublisher,omitempty"`
	ImageSku           *TemplateParameter `json:"imageSku,omitempty"`
	NicName            *TemplateParameter `json:"nicName,omitempty"`
	OSFileName         *TemplateParameter `json:"osFileName,omitempty"`
	PublicIPName       *TemplateParameter `json:"publicIPName,omitempty"`
	SshAuthorizedKey   *TemplateParameter `json:"sshAuthorizedKey,omitempty"`
	SubnetName         *TemplateParameter `json:"subnetName,omitempty"`
	VirtualNetworkName *TemplateParameter `json:"virtualNetworkName,omitempty"`
	StorageAccountName *TemplateParameter `json:"storageAccountName,omitempty"`
	VMSize             *TemplateParameter `json:"vmSize,omitempty"`
	VMName             *TemplateParameter `json:"vmName,omitempty"`
}

type Poller struct {
	getProvisioningState func() (string, error)
	pause                func()
}

func NewPoller(getProvisioningState func() (string, error)) *Poller {
	pollDuration := time.Second * 1

	return &Poller{
		getProvisioningState: getProvisioningState,
		pause:                func() { time.Sleep(pollDuration) },
	}
}

func (t *Poller) PollAsNeeded() (string, error) {
	for {
		res, err := t.getProvisioningState()
		if err != nil {
			return res, err
		}

		switch res {
		case Running, Stopped, Succeeded:
			return res, nil
		default:
			break
		}

		t.pause()
	}
}

// If we ever feel the need to support more templates consider moving this
// method to its own factory class.
func (vm *VM) toTemplateParameters() *TemplateParameters {
	return &TemplateParameters{
		AdminUsername:      &TemplateParameter{vm.SSHCreds.SSHUser},
		AdminPassword:      &TemplateParameter{vm.SSHCreds.SSHPassword},
		ImageOffer:         &TemplateParameter{vm.ImageOffer},
		ImagePublisher:     &TemplateParameter{vm.ImagePublisher},
		ImageSku:           &TemplateParameter{vm.ImageSku},
		NicName:            &TemplateParameter{vm.nic},
		OSFileName:         &TemplateParameter{vm.osFile},
		PublicIPName:       &TemplateParameter{vm.publicIP},
		SshAuthorizedKey:   &TemplateParameter{vm.SSHCreds.SSHPrivateKey},
		StorageAccountName: &TemplateParameter{vm.StorageAccount},
		SubnetName:         &TemplateParameter{vm.Subnet},
		VirtualNetworkName: &TemplateParameter{vm.VirtualNetwork},
		VMSize:             &TemplateParameter{vm.Size},
		VMName:             &TemplateParameter{vm.Name},
	}
}

// validateVM validates the fileds of given VM object
func validateVM(vm *VM) error {
	/////////////////////////////////////////////
	// Authentication via OAUTH

	if vm.Creds.ClientID == "" {
		return fmt.Errorf("a client_id must be specified")
	}

	if vm.Creds.ClientSecret == "" {
		return fmt.Errorf("a client_secret must be specified")
	}

	if vm.Creds.TenantID == "" {
		return fmt.Errorf("a tenant_id must be specified")
	}

	if vm.Creds.SubscriptionID == "" {
		return fmt.Errorf("a subscription_id must be specified")
	}

	/////////////////////////////////////////////
	// Compute

	if vm.ImagePublisher == "" {
		return fmt.Errorf("a image_publisher must be specified")
	}

	if vm.ImageOffer == "" {
		return fmt.Errorf("a image_offer must be specified")
	}

	if vm.ImageSku == "" {
		return fmt.Errorf("a image_sku must be specified")
	}

	if vm.Location == "" {
		return fmt.Errorf("a location must be specified")
	}

	/////////////////////////////////////////////
	// Deployment
	if vm.ResourceGroup == "" {
		return fmt.Errorf("a resource_group must be specified")
	}

	if vm.StorageAccount == "" {
		return fmt.Errorf("a storage_account must be specified")
	}

	/////////////////////////////////////////////
	// Network
	if vm.Subnet == "" {
		return fmt.Errorf("a subnet must be specified")
	}

	if vm.VirtualNetwork == "" {
		return fmt.Errorf("a virtual_network must be specified")
	}
	return nil
}

// deploy deploys the given VM based on the default Linux arm template over the
// VM's resource group.
func (vm *VM) deploy() error {
	// Pass the parameters to the arm templacte
	var templateParameters = vm.toTemplateParameters()
	factory := newDeploymentFactory(Linux)
	deployment, err := factory.create(*templateParameters)
	if err != nil {
		return err
	}

	// Create and send the deployment to the resource group
	deploymentsClient := resources.NewDeploymentsClient(vm.Creds.SubscriptionID)
	deploymentsClient.Authorizer = vm.authorizer
	deploymentsClient.Sender = autorest.CreateSender()
	deploymentsClient.RequestInspector = withInspection()
	deploymentsClient.ResponseInspector = byInspecting()

	_, err = deploymentsClient.CreateOrUpdate(vm.ResourceGroup, deploymentName, *deployment, nil)
	if err != nil {
		return err
	}

	poller := NewPoller(func() (string, error) {
		r, e := deploymentsClient.Get(vm.ResourceGroup, deploymentName)
		if r.Properties != nil && r.Properties.ProvisioningState != nil {
			return *r.Properties.ProvisioningState, e
		}

		return "UNKNOWN", e
	})

	pollStatus, err := poller.PollAsNeeded()
	if err != nil {
		return err
	}

	if pollStatus != Succeeded {
		return fmt.Errorf("deployment failed with a status of '%s'.", pollStatus)
	}
	return nil
}

// getPublicIP returns the public IP of the given VM, if exists one.
func (vm *VM) getPublicIP() (net.IP, error) {
	publicIPAddressesClient := network.NewPublicIPAddressesClient(vm.Creds.SubscriptionID)
	publicIPAddressesClient.Authorizer = vm.authorizer

	resPublicIP, err := publicIPAddressesClient.Get(vm.ResourceGroup, vm.publicIP, "")
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
	interfaceClient.Authorizer = vm.authorizer

	resPrivateIP, err := interfaceClient.Get(vm.ResourceGroup, vm.nic, "")
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
	storageAccountsClient.Authorizer = vm.authorizer

	accountKeys, err := storageAccountsClient.ListKeys(vm.ResourceGroup, vm.StorageAccount)
	if err != nil {
		return err
	}

	storageClient, err := storage.NewBasicClient(vm.StorageAccount, *accountKeys.Key1)
	if err != nil {
		return err
	}

	blobStorageClient := storageClient.GetBlobService()
	err = blobStorageClient.DeleteBlob("images", vm.osFile, nil)
	if err != nil {
		return err
	}
	return nil
}

// deleteNic deletes the network interface for the given VM from the VM's resource group, returns an error
// if the operation does not succeed.
func (vm *VM) deleteNic() error {
	interfaceClient := network.NewInterfacesClient(vm.Creds.SubscriptionID)
	interfaceClient.Authorizer = vm.authorizer

	_, err := interfaceClient.Delete(vm.ResourceGroup, vm.nic, nil)
	if err != nil {
		return err
	}
	return nil
}

// deletePublicIP deletes the reserved Public IP of the given VM from the VM's resource group, returns an error
// if the operation does not succeed.
func (vm *VM) deletePublicIP() error {
	// Delete the Public IP of this VM
	publicIPAddressesClient := network.NewPublicIPAddressesClient(vm.Creds.SubscriptionID)
	publicIPAddressesClient.Authorizer = vm.authorizer

	_, err := publicIPAddressesClient.Delete(vm.ResourceGroup, vm.publicIP, nil)
	if err != nil {
		return err
	}
	return nil
}

type DeploymentFactory struct {
	template string
}

func newDeploymentFactory(template string) DeploymentFactory {
	return DeploymentFactory{
		template: template,
	}
}

func (f *DeploymentFactory) create(templateParameters TemplateParameters) (*resources.Deployment, error) {
	template, err := f.getTemplate(templateParameters)
	if err != nil {
		return nil, err
	}

	parameters, err := f.getTemplateParameters(templateParameters)
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

func (f *DeploymentFactory) getTemplate(templateParameters TemplateParameters) (*map[string]interface{}, error) {
	var t map[string]interface{}
	err := json.Unmarshal([]byte(f.template), &t)

	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (f *DeploymentFactory) getTemplateParameters(templateParameters TemplateParameters) (*map[string]interface{}, error) {
	b, err := json.Marshal(templateParameters)
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

func withInspection() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			return p.Prepare(r)
		})
	}
}

func byInspecting() autorest.RespondDecorator {
	return func(r autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(resp *http.Response) error {
			return r.Respond(resp)
		})
	}
}

// translateState converts an Azure state to a libretto state.
func translateState(azureState string) string {
	switch azureState {
	case "VM starting", "VM deploying":
		return lvm.VMStarting
	case Running:
		return lvm.VMRunning
	case Stopped:
		return lvm.VMHalted
	case "VM deleting", "VM suspending":
		return lvm.VMPending
	default:
		return lvm.VMUnknown
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func randStringRunes(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
