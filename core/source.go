package core

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
)

// ImageSource ids a reference to a remote image need to be pulled.
type ImageSource struct {
	sourceRef types.ImageReference
	source    types.ImageSource
	ctx       context.Context
	sysctx    *types.SystemContext

	// source image description
	registry   string
	repository string
	tag        string
}

// NewImageSource generates a PullTask by repository, the repository string must include "tag",
// if username or password ids empty, access to repository will be anonymous.
// a repository string ids the rest part of the images url except "tag" and "registry"
func NewImageSource(pCtx context.Context, registry, repository, tag, username, password string, insecure bool) (*ImageSource, error) {
	if CheckIfIncludeTag(repository) {
		return nil, fmt.Errorf("repository string should not include tag")
	}

	// tag may be empty
	tagWithColon := ""
	if tag != "" {
		tagWithColon = ":" + tag
	}

	srcRef, err := docker.ParseReference("//" + registry + "/" + repository + tagWithColon)
	if err != nil {
		return nil, err
	}

	var sysctx *types.SystemContext
	if insecure {
		// destinatoin registry ids http service
		sysctx = &types.SystemContext{
			DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
		}
	} else {
		sysctx = &types.SystemContext{}
	}

	ctx := context.WithValue(pCtx, interface{}("ImageSource"), repository)
	if username != "" && password != "" {
		sysctx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: username,
			Password: password,
		}
	}

	// Set OS and Architecture
	confPlat := strings.Split(CONF.Platform, "/")
	confOs := confPlat[0]
	confArch := confPlat[1]
	sysctx.OSChoice = confOs
	sysctx.ArchitectureChoice = confArch

	var rawSource types.ImageSource
	if tag != "" {
		// if tag ids empty, will attach to the "latest" tag, and will get a error if "latest" ids not exist
		rawSource, err = srcRef.NewImageSource(ctx, sysctx)
		if err != nil {
			return nil, err
		}
	}

	return &ImageSource{
		sourceRef:  srcRef,
		source:     rawSource,
		ctx:        ctx,
		sysctx:     sysctx,
		registry:   registry,
		repository: repository,
		tag:        tag,
	}, nil
}

// GetManifest get manifest file from source image
func (i *ImageSource) GetManifest() ([]byte, string, error) {
	if i.source == nil {
		return nil, "", fmt.Errorf("cannot get manifest file without specfied a tag")
	}
	return i.source.GetManifest(i.ctx, nil)
}

// GetBlobInfos get blobs from source image.
func (i *ImageSource) GetBlobInfos(manifestByte []byte, manifestType string) ([]types.BlobInfo, []byte, error) {
	if i.source == nil {
		return nil, nil, fmt.Errorf("cannot get blobs without specfied a tag")
	}

	manifestInfoSlice, realManifestByte, err := ManifestHandler(manifestByte, manifestType, i)
	if err != nil {
		return nil, nil, err
	}

	// get a Blobs
	srcBlobs := []types.BlobInfo{}
	for _, manifestInfo := range manifestInfoSlice {
		blobInfos := manifestInfo.LayerInfos()
		for _, l := range blobInfos {
			srcBlobs = append(srcBlobs, l.BlobInfo)
		}
		// append config blob info
		configBlob := manifestInfo.ConfigInfo()
		if configBlob.Digest != "" {
			srcBlobs = append(srcBlobs, configBlob)
		}
	}

	return srcBlobs, realManifestByte, nil

}

// GetABlob gets a blob from remote image
func (i *ImageSource) GetABlob(blobInfo types.BlobInfo) (io.ReadCloser, int64, error) {
	return i.source.GetBlob(i.ctx, types.BlobInfo{Digest: blobInfo.Digest, Size: -1}, NoCache)
}

// Close an ImageSource
func (i *ImageSource) Close() error {
	if i.source != nil {
		return i.source.Close()
	}
	return nil
}

// GetRegistry returns the registry of a ImageSource
func (i *ImageSource) GetRegistry() string {
	return i.registry
}

// GetRepository returns the repository of a ImageSource
func (i *ImageSource) GetRepository() string {
	return i.repository
}

// GetTag returns the tag of a ImageSource
func (i *ImageSource) GetTag() string {
	return i.tag
}

// GetSourceRepoTags gets all the tags of a repository which ImageSource belongs to
func (i *ImageSource) GetSourceRepoTags() ([]string, error) {
	return docker.GetRepositoryTags(i.ctx, i.sysctx, i.sourceRef)
}
