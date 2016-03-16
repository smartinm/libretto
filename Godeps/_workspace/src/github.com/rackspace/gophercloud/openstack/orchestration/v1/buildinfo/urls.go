package buildinfo

import "github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud"

func getURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("build_info")
}
