package glusterfs

import (
	"context"
	"strings"

	"github.com/NativeCI/gluster-heketi-csi-driver/util"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"github.com/heketi/heketi/pkg/glusterfs/api"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Empty = csi.UnimplementedControllerServer

type ControllerServer struct {
	*GfDriver
	*Empty
}

func NewControllerServer(g *GfDriver) *ControllerServer {
	return &ControllerServer{
		GfDriver: g,
	}
}

const (
	volumeOwnerAnn          = "VolumeOwner"
	defaultVolumeSize   int = 1 // default volume size ie 1 GB
	defaultReplicaCount     = 3
	defaultBrickType        = "lvm"
	brickTypeLoop           = "loop"
	brickTypeLvm            = "lvm"
)

// Run start a non-blocking grpc controller,node and identityserver for
// GlusterFS CSI driver which can serve multiple parallel requests
func (server *ControllerServer) Run() {
	srv := csicommon.NewNonBlockingGRPCServer()
	srv.Start(server.Endpoint, NewIdentityServer(server.GfDriver), server, nil)
	srv.Wait()
}

// CreateVolume creates and starts the volume
func (cs *ControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	glog.V(2).Infof("request received %+v", protosanitizer.StripSecrets(req))

	glog.V(1).Infof("creating volume with name %s", req.Name)

	volSizeBytes := cs.getVolumeSize(req)

	// If volume does not exist, provision volume
	//Create the volume
	volume, err := cs.client.VolumeCreate(&api.VolumeCreateRequest{
		Size: volSizeBytes,
		Name: req.Name,
	})
	if err != nil {
		glog.Errorf("failed to create volume: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to create volume: %v", err)
	}

	resp := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volume.Id,
			CapacityBytes: req.GetCapacityRange().RequiredBytes,
			VolumeContext: map[string]string{
				"glustervol":        volumeName,
				"glusterserver":     glusterServer,
				"glusterbkpservers": strings.Join(bkpServers, ":"),
			},
		},
	}

	glog.V(4).Infof("CSI volume response: %+v", protosanitizer.StripSecrets(resp))
	return resp, nil
}

func (cs *ControllerServer) getVolumeSize(req *csi.CreateVolumeRequest) int {
	// If capacity mentioned, pick that or use default size 1 GB
	volSizeBytes := defaultVolumeSize
	if capRange := req.GetCapacityRange(); capRange != nil {
		volSizeBytes = util.FromBytesToGb(capRange.GetRequiredBytes())
	}
	return volSizeBytes
}
