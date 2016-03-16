package serviceassets

import "github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud"

func deleteURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL("services", id, "assets")
}
