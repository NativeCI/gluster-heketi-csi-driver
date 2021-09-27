package glusterfs

import (
	"context"
	"errors"
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

type VolumeCapacity interface {
	GetCapacityRange() *csi.CapacityRange
}

type ControllerServer struct {
	*GfDriver
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
	//TODO create volume based on the request input, like distribute, replica etc.
	volume, err := cs.client.VolumeCreate(&api.VolumeCreateRequest{
		Size: volSizeBytes,
		Name: req.Name,
	})
	if err != nil {
		glog.Errorf("failed to create volume: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to create volume: %v", err)
	}
	glusterServer, bkpServers, err := cs.getClusterNodes()
	if err != nil {
		glog.Errorf("failed to get cluster nodes: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get cluster nodes: %v", err)
	}

	resp := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volume.Id,
			CapacityBytes: req.GetCapacityRange().RequiredBytes,
			VolumeContext: map[string]string{
				"glustervol":        volume.Id,
				"glusterserver":     *glusterServer,
				"glusterbkpservers": strings.Join(bkpServers, ":"),
			},
		},
	}

	glog.V(4).Infof("CSI volume response: %+v", protosanitizer.StripSecrets(resp))
	return resp, nil
}

func (cs *ControllerServer) getVolumeSize(req VolumeCapacity) int {
	// If capacity mentioned, pick that or use default size 1 GB
	volSizeBytes := defaultVolumeSize
	if capRange := req.GetCapacityRange(); capRange != nil {
		volSizeBytes = util.FromBytesToGb(capRange.GetRequiredBytes())
	}
	return volSizeBytes
}

func (cs *ControllerServer) getMainCluster() (*string, error) {
	clusters, err := cs.client.ClusterList()
	if err != nil {
		return nil, err
	}
	if len(clusters.Clusters) == 0 {
		return nil, errors.New("No clusters available")
	}
	clusterID := clusters.Clusters[0]
	return &clusterID, nil
}

func (cs *ControllerServer) getClusterNodes() (*string, []string, error) {
	clusterID, err := cs.getMainCluster()
	if err != nil {
		return nil, nil, err
	}
	cluster, err := cs.client.ClusterInfo(*clusterID)
	if err != nil {
		return nil, nil, err
	}
	glusterServer := ""
	bkpservers := []string{}

	for i, p := range cluster.Nodes {
		node, err := cs.client.NodeInfo(p)
		if err != nil {
			continue
		}
		if i == 0 {
			glusterServer = node.Hostnames.Storage[0]
			continue
		}
		bkpservers = append(bkpservers, node.Hostnames.Storage...)
	}
	glog.V(2).Infof("primary and backup gluster servers [%+v,%+v]", glusterServer, bkpservers)

	return &glusterServer, bkpservers, err
}

// DeleteVolume deletes the given volume.
func (cs *ControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is nil")
	}
	glog.V(2).Infof("deleting volume with ID: %s", volumeID)
	err := cs.client.VolumeDelete(req.VolumeId)
	if err != nil {
		glog.Errorf("deleting volume %s failed: %v", req.VolumeId, err)
		return nil, status.Errorf(codes.Internal, "deleting volume %s failed: %v", req.VolumeId, err)
	}
	glog.Infof("successfully deleted volume %s", volumeID)
	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerPublishVolume return Unimplemented error
func (cs *ControllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerUnpublishVolume return Unimplemented error
func (cs *ControllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ValidateVolumeCapabilities checks whether the volume capabilities requested
// are supported.
func (cs *ControllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "ValidateVolumeCapabilities() - request is nil")
	}

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "ValidateVolumeCapabilities() - VolumeId is nil")
	}

	reqCaps := req.GetVolumeCapabilities()
	if reqCaps == nil {
		return nil, status.Error(codes.InvalidArgument, "ValidateVolumeCapabilities() - VolumeCapabilities is nil")
	}

	_, err := cs.client.VolumeInfo(volumeID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "ValidateVolumeCapabilities() - %v", err)
	}

	var vcaps []*csi.VolumeCapability_AccessMode
	for _, mode := range []csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	} {
		vcaps = append(vcaps, &csi.VolumeCapability_AccessMode{Mode: mode})
	}
	capSupport := false

	for _, cap := range reqCaps {
		for _, m := range vcaps {
			if m.Mode == cap.AccessMode.Mode {
				capSupport = true
			}
		}
	}

	if !capSupport {
		return nil, status.Errorf(codes.NotFound, "%v not supported", req.GetVolumeCapabilities())
	}

	resp := &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: req.VolumeCapabilities,
		},
	}

	glog.V(1).Infof("GlusterFS CSI driver volume capabilities: %+v", resp)
	return resp, nil
}

// ListVolumes returns a list of volumes
func (cs *ControllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	// Fetch all the volumes in the TSP
	volumes, err := cs.client.VolumeList()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	var entries []*csi.ListVolumesResponse_Entry
	for _, vol := range volumes.Volumes {
		volume, err := cs.client.VolumeInfo(vol)
		if err != nil {
			return nil, err
		}
		entries = append(entries, &csi.ListVolumesResponse_Entry{
			Volume: &csi.Volume{
				VolumeId:      volume.Id,
				CapacityBytes: util.FromGbToBytes(int64(volume.Size)),
			},
		})
	}

	resp := &csi.ListVolumesResponse{
		Entries: entries,
	}

	return resp, nil
}

// GetCapacity returns the capacity of the storage pool
func (cs *ControllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerGetCapabilities returns the capabilities of the controller service.
func (cs *ControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	newCap := func(cap csi.ControllerServiceCapability_RPC_Type) *csi.ControllerServiceCapability {
		return &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: cap,
				},
			},
		}
	}

	var caps []*csi.ControllerServiceCapability
	for _, cap := range []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
	} {
		caps = append(caps, newCap(cap))
	}

	resp := &csi.ControllerGetCapabilitiesResponse{
		Capabilities: caps,
	}

	return resp, nil
}

func (cs *ControllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateSnapshot not implemented")
}
func (cs *ControllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteSnapshot not implemented")
}
func (cs *ControllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListSnapshots not implemented")
}

func (cs *ControllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	glog.V(2).Infof("request received %+v", protosanitizer.StripSecrets(req))
	glog.V(1).Infof("expanding volume with name %s", req.VolumeId)

	volSizeBytes := cs.getVolumeSize(req)
	volume, err := cs.client.VolumeExpand(req.VolumeId, &api.VolumeExpandRequest{
		Size: util.FromBytesToGb(int64(volSizeBytes)),
	})
	if err != nil {
		glog.Errorf("failed to expand volume: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to expand volume: %v", err)
	}
	return &csi.ControllerExpandVolumeResponse{
		CapacityBytes:         util.FromGbToBytes(int64(volume.Size)),
		NodeExpansionRequired: true,
	}, nil
}
func (cs *ControllerServer) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	glog.V(2).Infof("request received %+v", protosanitizer.StripSecrets(req))
	glog.V(1).Infof("getting volume with name %s", req.VolumeId)
	volume, err := cs.client.VolumeInfo(req.VolumeId)
	if err != nil {
		glog.Errorf("failed to get volume: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get volume: %v", err)
	}
	glusterServer, bkpServers, err := cs.getClusterNodes()
	if err != nil {
		glog.Errorf("failed to get cluster nodes: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get cluster nodes: %v", err)
	}

	return &csi.ControllerGetVolumeResponse{
		Volume: &csi.Volume{
			CapacityBytes: util.FromGbToBytes(int64(volume.Size)),
			VolumeId:      volume.Id,
			VolumeContext: map[string]string{
				"glustervol":        volume.Id,
				"glusterserver":     *glusterServer,
				"glusterbkpservers": strings.Join(bkpServers, ":"),
			},
		},
	}, nil
}
