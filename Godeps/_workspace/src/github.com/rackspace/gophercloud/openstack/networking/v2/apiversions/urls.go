package apiversions

import (
	"strings"

	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud"
)

func apiVersionsURL(c *gophercloud.ServiceClient) string {
	return c.Endpoint
}

func apiInfoURL(c *gophercloud.ServiceClient, version string) string {
	return c.Endpoint + strings.TrimRight(version, "/") + "/"
}
