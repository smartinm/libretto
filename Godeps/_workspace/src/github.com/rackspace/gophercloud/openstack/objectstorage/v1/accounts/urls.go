package accounts

import "github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud"

func getURL(c *gophercloud.ServiceClient) string {
	return c.Endpoint
}

func updateURL(c *gophercloud.ServiceClient) string {
	return getURL(c)
}
