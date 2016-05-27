/*
* @Author: Lear Li
* @Date:   2016-05-27 11:10:34
* @Last Modified by:   Lear Li
* @Last Modified time: 2016-05-31 14:38:30
 */

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/apcera/libretto/ssh"
	"github.com/apcera/libretto/virtualmachine/google"
)

func main() {

	accountFile := "/Users/liruili/Downloads/account.json"

	sshPrivateKey, err := ioutil.ReadFile("/Users/liruili/.ssh/gce-ubuntu.pem")
	if err != nil {
		panic(err)
	}

	sshPublicKey, err := ioutil.ReadFile("/Users/liruili/.ssh/gce-ubuntu.pem.pub")
	if err != nil {
		panic(err)
	}

	scopes := []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/compute",
		"https://www.googleapis.com/auth/devstorage.full_control",
		"https://www.googleapis.com/auth/logging.write",
		"https://www.googleapis.com/auth/monitoring.write",
		"https://www.googleapis.com/auth/servicecontrol",
		"https://www.googleapis.com/auth/service.management",
	}

	vm := &google.VM{
		Name:          "libretto-vm-6",
		Zone:          "us-central1-a",
		MachineType:   "n1-standard-1",
		SourceImage:   "ubuntu-1404-trusty-v20160516",
		DiskType:      "pd-standard",
		DiskSize:      10,
		Preemptible:   false,
		Network:       "default",
		Subnetwork:    "default",
		UseInternalIP: false,
		Scopes:        scopes,
		Project:       "flamingo-kitty-devtest",

		AccountFile: accountFile,
		SSHCreds: ssh.Credentials{
			SSHUser:       "ubuntu",
			SSHPrivateKey: string(sshPrivateKey),
		},
		SSHPublicKey: string(sshPublicKey),
		Tags:         []string{"libretto"},
	}

	if err := vm.Provision(); err != nil {
		fmt.Println(err)
	}

	// Wait for instance to ready
	// time.Sleep(60 * time.Second)

	ips, err := vm.GetIPs()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Get IPs from GCE instance")
		fmt.Println(ips)
	}

	fmt.Println("Going to get ssh connection")

	option := ssh.Options{
		KeepAlive: 2,
	}

	client, err := vm.GetSSH(option)
	if err != nil {
		panic(err)
	}
	fmt.Println("Get SSH")

	defer client.Disconnect()

	err = client.Connect()
	if err != nil {
		panic(err)
	}

	content, err := ioutil.ReadFile("/Users/liruili/aws_cf_config.json")
	if err != nil {
		panic(err)
	}
	fmt.Println("Uploading file to instance")
	err = client.Upload(bytes.NewReader(content), "test.json", 0755)
	if err != nil {
		panic(err)
	}

	fmt.Println("stop vm")
	if err := vm.Halt(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("start vm")
	if err := vm.Start(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("Destroy vm")
	if err := vm.Destroy(); err != nil {
		fmt.Println(err)
	}

}
