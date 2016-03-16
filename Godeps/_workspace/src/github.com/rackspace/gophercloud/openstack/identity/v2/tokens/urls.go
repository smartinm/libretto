package tokens

import "github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud"

// CreateURL generates the URL used to create new Tokens.
func CreateURL(client *gophercloud.ServiceClient) string {
	return client.ServiceURL("tokens")
}
