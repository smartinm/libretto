package cdnobjects

import (
	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud"
	os "github.com/apcera/libretto/Godeps/_workspace/src/github.com/rackspace/gophercloud/openstack/objectstorage/v1/objects"
)

// Delete is a function that deletes an object from the CDN.
func Delete(c *gophercloud.ServiceClient, containerName, objectName string, opts os.DeleteOptsBuilder) os.DeleteResult {
	return os.Delete(c, containerName, objectName, nil)
}
