package podman

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

func ContainerList(ctx context.Context, options dockerctr.ListOptions) ([]dockertypes.Container, error) {
	return Client.ContainerList(ctx, options)
}

func ContainerInspect(ctx context.Context, containerID string) (dockertypes.ContainerJSON, error) {
	return Client.ContainerInspect(ctx, containerID)
}

func ContainerStats(ctx context.Context, containerID string, stream bool) (dockerctr.StatsResponseReader, error) {
	return Client.ContainerStats(ctx, containerID, stream)
}

func ContainerLogs(ctx context.Context, containerID string, options dockerctr.LogsOptions) (io.ReadCloser, error) {
	return Client.ContainerLogs(ctx, containerID, options)
}

func ContainerStop(ctx context.Context, containerID string, options dockerctr.StopOptions) error {
	return Client.ContainerStop(ctx, containerID, options)
}

func ContainerRemove(ctx context.Context, containerID string, options dockerctr.RemoveOptions) error {
	return Client.ContainerRemove(ctx, containerID, options)
}

func ContainerStart(ctx context.Context, containerID string, options dockerctr.StartOptions) error {
	return Client.ContainerStart(ctx, containerID, options)
}

func ContainerRestart(ctx context.Context, containerID string, options dockerctr.StopOptions) error {
	return Client.ContainerRestart(ctx, containerID, options)
}

func ContainerCreate(ctx context.Context, config *dockerctr.Config, hostConfig *dockerctr.HostConfig, networkingConfig *dockernet.NetworkingConfig, platform *v1.Platform, containerName string) (dockerctr.CreateResponse, error) {
	return Client.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName)
}

func NetworkList(ctx context.Context, options dockernet.ListOptions) ([]dockernet.Inspect, error) {
	return Client.NetworkList(ctx, options)
}

func ImageList(ctx context.Context, options dockerimage.ListOptions) ([]dockerimage.Summary, error) {
	return Client.ImageList(ctx, options)
}

// LocalImageIDs returns a snapshot of locally cached images, keyed by fully
// qualified reference ("docker.io/library/alpine:latest") with the image ID as
// the value. Taking two snapshots around a pull and diffing them yields a
// structural, provider-independent answer to "did anything actually change?",
// without parsing human-readable CLI output.
func LocalImageIDs(ctx context.Context) (map[string]string, error) {
	images, err := ImageList(ctx, dockerimage.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(images))
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == "" || tag == "<none>:<none>" {
				continue
			}
			out[tag] = img.ID
		}
	}
	return out, nil
}

func ImageInspectWithRaw(ctx context.Context, imageID string) (dockertypes.ImageInspect, []byte, error) {
	return Client.ImageInspectWithRaw(ctx, imageID)
}

func ImagePull(ctx context.Context, refStr string, options dockerimage.PullOptions) (io.ReadCloser, error) {
	return Client.ImagePull(ctx, refStr, options)
}

func ImageRemove(ctx context.Context, imageID string, options dockerimage.RemoveOptions) ([]dockerimage.DeleteResponse, error) {
	return Client.ImageRemove(ctx, imageID, options)
}

func ImagesPrune(ctx context.Context, filters dockerfilters.Args) (dockerimage.PruneReport, error) {
	return Client.ImagesPrune(ctx, filters)
}

func Info(ctx context.Context) (system.Info, error) {
	return Client.Info(ctx)
}

func VolumesPrune(ctx context.Context, filters dockerfilters.Args) (dockervol.PruneReport, error) {
	return Client.VolumesPrune(ctx, filters)
}

func VolumeList(ctx context.Context, options dockervol.ListOptions) (dockervol.ListResponse, error) {
	return Client.VolumeList(ctx, options)
}

func VolumeInspect(ctx context.Context, volumeID string) (dockervol.Volume, error) {
	return Client.VolumeInspect(ctx, volumeID)
}

func VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	return Client.VolumeRemove(ctx, volumeID, force)
}

func DiskUsage(ctx context.Context, options dockertypes.DiskUsageOptions) (dockertypes.DiskUsage, error) {
	return Client.DiskUsage(ctx, options)
}
