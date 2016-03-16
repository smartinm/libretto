package roles

import "github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud"

func listAssignmentsURL(client *gophercloud.ServiceClient) string {
	return client.ServiceURL("role_assignments")
}
