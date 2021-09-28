package glusterfs

import (
	"github.com/NativeCI/gluster-heketi-csi-driver/config"
	heketi "github.com/heketi/heketi/client/api/go-client"
	log "github.com/sirupsen/logrus"
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
		log.Errorf("GlusterFS CSI driver initialization failed")
		return nil
	}

	gfd.Config = config
	gfd.client = heketi.NewClient(gfd.Config.HeketiURL, gfd.Config.HeketiUser, gfd.Config.HeketiSecret)

	log.Infof("GlusterFS CSI driver initialized")

	return gfd
}
