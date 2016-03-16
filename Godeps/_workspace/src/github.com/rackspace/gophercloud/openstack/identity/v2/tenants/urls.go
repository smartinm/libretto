package tenants

import "github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud"

func listURL(client *gophercloud.ServiceClient) string {
	return client.ServiceURL("tenants")
}
