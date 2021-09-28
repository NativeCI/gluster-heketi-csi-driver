package glusterfs

import (
	"context"
	"os"
	"strings"

	"github.com/NativeCI/gluster-heketi-csi-driver/util"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/mount-utils"
)

// NodeServer struct of Glusterfs CSI driver with supported methods of CSI node
// server spec.
type NodeServer struct {
	*GfDriver
}

// NewNodeServer initialize a node server for GlusterFS CSI driver.
func NewNodeServer(g *GfDriver) *NodeServer {
	return &NodeServer{
		GfDriver: g,
	}
}

// Run start a non-blocking grpc controller,node and identityserver for
// GlusterFS CSI driver which can serve multiple parallel requests
func (server *NodeServer) Run() {
	srv := csicommon.NewNonBlockingGRPCServer()
	srv.Start(server.Endpoint, NewIdentityServer(server.GfDriver), nil, server)
	srv.Wait()
}

var glusterMounter = mount.New("")

// NodeStageVolume mounts the volume to a staging path on the node.
func (ns *NodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeUnstageVolume unstages the volume from the staging path
func (ns *NodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodePublishVolume mounts the volume mounted to the staging path to the target
// path
func (ns *NodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	log.Infof("received node publish volume request %+v", protosanitizer.StripSecrets(req))

	if err := ns.validateNodePublishVolumeReq(req); err != nil {
		return nil, err
	}

	targetPath := req.GetTargetPath()

	notMnt, err := glusterMounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			// #nosec
			log.Infof("creating a new directory at %s", targetPath)
			if err = os.MkdirAll(targetPath, 0777); err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			notMnt = true
		} else {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	if !notMnt {
		return &csi.NodePublishVolumeResponse{}, nil
	}

	mo := req.GetVolumeCapability().GetMount().GetMountFlags()

	if req.GetReadonly() {
		mo = append(mo, "ro")
	}
	source := req.GetVolumeContext()["glustermountpoint"]
	err = doMount(source, targetPath, mo)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func doMount(source, targetPath string, mo []string) error {
	err := glusterMounter.Mount(source, targetPath, "glusterfs", mo)
	if err != nil {
		if os.IsPermission(err) {
			return status.Error(codes.PermissionDenied, err.Error())
		}
		if strings.Contains(err.Error(), "invalid argument") {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return status.Error(codes.Internal, err.Error())
	}
	// #nosec
	err = os.Chmod(targetPath, 0777)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	return nil
}

func (ns *NodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {

	// TODO need to implement volume status call
	return nil, status.Error(codes.Unimplemented, "")

}

func (ns *NodeServer) validateNodePublishVolumeReq(req *csi.NodePublishVolumeRequest) error {
	if req == nil {
		return status.Error(codes.InvalidArgument, "request cannot be empty")
	}

	if req.GetVolumeId() == "" {
		return status.Error(codes.InvalidArgument, "NodePublishVolume Volume ID must be provided")
	}

	if req.GetTargetPath() == "" {
		return status.Error(codes.InvalidArgument, "NodePublishVolume Target Path cannot be empty")
	}

	if req.GetVolumeCapability() == nil {
		return status.Error(codes.InvalidArgument, "NodePublishVolume Volume Capability must be provided")
	}
	return nil
}

// NodeUnpublishVolume unmounts the volume from the target path
func (ns *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request cannot be empty")
	}

	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume Volume ID must be provided")
	}

	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume Target Path must be provided")
	}

	targetPath := req.GetTargetPath()
	notMnt, err := glusterMounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, status.Error(codes.NotFound, "targetpath not found")
		}
		return nil, status.Error(codes.Internal, err.Error())

	}

	if notMnt {
		return nil, status.Error(codes.NotFound, "volume not mounted")
	}

	err = util.UnmountPath(req.GetTargetPath(), glusterMounter)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeGetInfo returns NodeGetInfoResponse for CO.
func (ns *NodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId: ns.GfDriver.NodeID,
	}, nil
}

// NodeGetCapabilities returns the supported capabilities of the node server
func (ns *NodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{}, nil
}

func (ns *NodeServer) NodeExpandVolume(context.Context, *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return &csi.NodeExpandVolumeResponse{}, nil
}
