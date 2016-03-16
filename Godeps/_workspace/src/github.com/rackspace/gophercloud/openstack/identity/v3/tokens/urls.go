package tokens

import "github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud"

func tokenURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("auth", "tokens")
}
