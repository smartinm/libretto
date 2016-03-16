package bootfromvolume

import "github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud"

func createURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("os-volumes_boot")
}
