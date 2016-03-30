// Copyright 2016 Apcera Inc. All rights reserved.

package main

import (
	"fmt"

	"github.com/apcera/libretto/virtualmachine/azureV2"
)

func main() {
	/*vm := &azureV2.VM{
		PublishSettings:  os.Getenv("AZURE_PUBLISH_SETTING_FILE"),
		ServiceName:      "libretto-sample",
		Label:            "libretto-sample",
		Name:             "libretto-sample",
		Size:             "Medium",
		SourceImage:      os.Getenv("AZURE_IMAGE_NAME"),
		StorageAccount:   os.Getenv("AZURE_STORAGE_ACCOUNT"),
		StorageContainer: os.Getenv("AZURE_STORAGE_CONTAINER"),
		Location:         "Central US",
		ConfigureHTTP:    true,
		SSHCreds: ssh.Credentials{
			SSHUser:     "ubuntu",
			SSHPassword: "Ubuntu123",
		},
		DeployOptions: azureV2.DeploymentOptions{
			VirtualNetworkName: os.Getenv("AZURE_VIRTUAL_NETWORK"),
		},
	}*/

	fmt.Println("Checking Name")
	azureV2.CheckName("packertest1")
	/*fmt.Println("provisioning...")
	err := vm.Provision()
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("getting IP Addresses...")
	ips := vm.GetIPs()
	fmt.Println("IP Addresses: %v", ips)

	fmt.Println("creating ssh connection...")
	cli, err := vm.GetSSH(ssh.Options{KeepAlive: 2})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("vm ssh:", cli)

	fmt.Println("getting status...")
	state, err := vm.GetState()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("vm state:", state)

	fmt.Println("shutting down vm...")
	err = vm.Halt()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("getting status...")
	state, err = vm.GetState()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("starting vm...")
	err = vm.Start()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("getting status...")
	state, err = vm.GetState()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("vm state:", state)

	fmt.Println("deleting vm...")
	err = vm.Destroy()
	if err != nil {
		log.Fatal(err)
	}*/
}
