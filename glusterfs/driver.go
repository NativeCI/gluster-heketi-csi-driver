package glusterfs

import (
	"github.com/NativeCI/gluster-heketi-csi-driver/config"
	"github.com/golang/glog"
	heketi "github.com/heketi/heketi/client/api/go-client"
)

const (
	glusterfsCSIDriverName    = "org.gluster.glusterfs"
	glusterfsCSIDriverVersion = "1.0.0"
)

// GfDriver is the struct embedding information about the connection to gluster
// cluster and configuration of CSI driver.
type GfDriver struct {
	client *heketi.Client
	*config.Config
}

// New returns CSI driver
func New(config *config.Config) *GfDriver {
	gfd := &GfDriver{}

	if config == nil {
		glog.Errorf("GlusterFS CSI driver initialization failed")
		return nil
	}

	gfd.Config = config
	gfd.client = heketi.NewClient(gfd.Config.HeketiURL, gfd.Config.HeketiUser, gfd.Config.HeketiSecret)

	glog.V(1).Infof("GlusterFS CSI driver initialized")

	return gfd
}
