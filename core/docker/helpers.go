package docker

import (
	"context"
	"io"

	dockertypes "github.com/docker/docker/api/types"
	dockerctr "github.com/docker/docker/api/types/container"
	dockerfilters "github.com/docker/docker/api/types/filters"
	dockerimage "github.com/docker/docker/api/types/image"
	dockernet "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	dockervol "github.com/docker/docker/api/types/volume"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// ContainerList wraps the Docker API to list containers.
func ContainerList(ctx context.Context, options dockerctr.ListOptions) ([]dockertypes.Container, error) {
	return Client.ContainerList(ctx, options)
}

// ContainerInspect wraps the Docker API to inspect a container.
func ContainerInspect(ctx context.Context, containerID string) (dockertypes.ContainerJSON, error) {
	return Client.ContainerInspect(ctx, containerID)
}

// ContainerStats wraps the Docker API to get container stats.
func ContainerStats(ctx context.Context, containerID string, stream bool) (dockerctr.StatsResponseReader, error) {
	return Client.ContainerStats(ctx, containerID, stream)
}

// ContainerLogs wraps the Docker API to get container logs.
func ContainerLogs(ctx context.Context, containerID string, options dockerctr.LogsOptions) (io.ReadCloser, error) {
	return Client.ContainerLogs(ctx, containerID, options)
}

// ContainerStop wraps the Docker API to stop a container.
func ContainerStop(ctx context.Context, containerID string, options dockerctr.StopOptions) error {
	return Client.ContainerStop(ctx, containerID, options)
}

// ContainerRemove wraps the Docker API to remove a container.
func ContainerRemove(ctx context.Context, containerID string, options dockerctr.RemoveOptions) error {
	return Client.ContainerRemove(ctx, containerID, options)
}

// ContainerStart wraps the Docker API to start a container.
func ContainerStart(ctx context.Context, containerID string, options dockerctr.StartOptions) error {
	return Client.ContainerStart(ctx, containerID, options)
}

// ContainerRestart wraps the Docker API to restart a container.
func ContainerRestart(ctx context.Context, containerID string, options dockerctr.StopOptions) error {
	return Client.ContainerRestart(ctx, containerID, options)
}

// ContainerCreate wraps the Docker API to create a container.
func ContainerCreate(ctx context.Context, config *dockerctr.Config, hostConfig *dockerctr.HostConfig, networkingConfig *dockernet.NetworkingConfig, platform *v1.Platform, containerName string) (dockerctr.CreateResponse, error) {
	return Client.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName)
}

// NetworkList wraps the Docker API to list networks.
func NetworkList(ctx context.Context, options dockernet.ListOptions) ([]dockernet.Inspect, error) {
	return Client.NetworkList(ctx, options)
}

// ImageList wraps the Docker API to list images.
func ImageList(ctx context.Context, options dockerimage.ListOptions) ([]dockerimage.Summary, error) {
	return Client.ImageList(ctx, options)
}

// ImageInspectWithRaw wraps the Docker API to inspect an image.
func ImageInspectWithRaw(ctx context.Context, imageID string) (dockertypes.ImageInspect, []byte, error) {
	return Client.ImageInspectWithRaw(ctx, imageID)
}

// ImageRemove wraps the Docker API to remove an image.
func ImageRemove(ctx context.Context, imageID string, options dockerimage.RemoveOptions) ([]dockerimage.DeleteResponse, error) {
	return Client.ImageRemove(ctx, imageID, options)
}

// ImagesPrune wraps the Docker API to prune images.
func ImagesPrune(ctx context.Context, filters dockerfilters.Args) (dockerimage.PruneReport, error) {
	return Client.ImagesPrune(ctx, filters)
}

// Info wraps the Docker API to get system info.
func Info(ctx context.Context) (system.Info, error) {
	return Client.Info(ctx)
}

// VolumesPrune wraps the Docker API to prune volumes.
func VolumesPrune(ctx context.Context, filters dockerfilters.Args) (dockervol.PruneReport, error) {
	return Client.VolumesPrune(ctx, filters)
}

// VolumeList wraps the Docker API to list volumes.
func VolumeList(ctx context.Context, options dockervol.ListOptions) (dockervol.ListResponse, error) {
	return Client.VolumeList(ctx, options)
}

// VolumeInspect wraps the Docker API to inspect a volume.
func VolumeInspect(ctx context.Context, volumeID string) (dockervol.Volume, error) {
	return Client.VolumeInspect(ctx, volumeID)
}

// VolumeRemove wraps the Docker API to remove a volume.
func VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	return Client.VolumeRemove(ctx, volumeID, force)
}

// DiskUsage wraps the Docker API to get disk usage.
func DiskUsage(ctx context.Context, options dockertypes.DiskUsageOptions) (dockertypes.DiskUsage, error) {
	return Client.DiskUsage(ctx, options)
}
