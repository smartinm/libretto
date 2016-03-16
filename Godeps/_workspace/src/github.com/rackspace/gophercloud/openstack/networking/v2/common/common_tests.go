package common

import (
	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud"
	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud/testhelper/client"
)

const TokenID = client.TokenID

func ServiceClient() *gophercloud.ServiceClient {
	sc := client.ServiceClient()
	sc.ResourceBase = sc.Endpoint + "v2.0/"
	return sc
}
