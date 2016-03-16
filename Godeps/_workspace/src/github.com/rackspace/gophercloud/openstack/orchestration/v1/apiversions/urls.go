package apiversions

import "github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud"

func apiVersionsURL(c *gophercloud.ServiceClient) string {
	return c.Endpoint
}
