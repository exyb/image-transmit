package core

import (
	"fmt"

	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/containers/image/v5/manifest"
)

// ManifestHandler expends the ability of handling manifest list in schema2, but it's not finished yet
// return the digest array of manifests in the manifest list if exist.
func ManifestHandler(m []byte, t string, i *ImageSource) ([]manifest.Manifest, []byte, string, error) {

	var manifestInfoSlice []manifest.Manifest

	if t == manifest.DockerV2Schema2MediaType {
		manifestInfo, err := manifest.Schema2FromManifest(m)
		if err != nil {
			return nil, nil, "", err
		}
		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)
		return manifestInfoSlice, m, manifest.DockerV2Schema2MediaType, nil
	} else if t == manifest.DockerV2Schema1MediaType || t == manifest.DockerV2Schema1SignedMediaType {
		manifestInfo, err := manifest.Schema1FromManifest(m)
		if err != nil {
			return nil, nil, "", err
		}
		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)
		return manifestInfoSlice, m, t, nil
	} else if t == manifest.DockerV2ListMediaType {
		manifestSchemaListInfo, err := manifest.Schema2ListFromManifest(m)
		if err != nil {
			return nil, nil, "", err
		}
		realManifestByte := m
		realManifestType := t
		for _, manifestDescriptorElem := range manifestSchemaListInfo.Manifests {

			manifestByte, manifestType, err := i.source.GetManifest(i.ctx, &manifestDescriptorElem.Digest)
			if err != nil {
				return nil, nil, "", err
			}
			if manifestDescriptorElem.Platform.OS != i.sysctx.OSChoice || manifestDescriptorElem.Platform.Architecture != i.sysctx.ArchitectureChoice {
				continue
			}
			platformSpecManifest, singleManifestByte, singleManifestType, err := ManifestHandler(manifestByte, manifestType, i)
			if err != nil {
				return nil, nil, "", err
			}

			manifestInfoSlice = append(manifestInfoSlice, platformSpecManifest...)
			realManifestByte = singleManifestByte
			realManifestType = singleManifestType
		}
		return manifestInfoSlice, realManifestByte, realManifestType, nil
	}

	if t == imgspecv1.MediaTypeImageManifest {
		manifestInfo, err := manifest.OCI1FromManifest(m)
		if err != nil {
			return nil, m, "", err
		}
		manifestInfoSlice = append(manifestInfoSlice, manifestInfo)
		return manifestInfoSlice, m, imgspecv1.MediaTypeImageManifest, nil
	} else if t == imgspecv1.MediaTypeImageIndex {
		manifestSchemaListInfo, err := manifest.OCI1IndexFromManifest(m)
		if err != nil {
			return nil, nil, "", err
		}
		realManifestByte := m
		realManifestType := t

		for _, manifestDescriptorElem := range manifestSchemaListInfo.Manifests {

			manifestByte, manifestType, err := i.source.GetManifest(i.ctx, &manifestDescriptorElem.Digest)
			if err != nil {
				return nil, nil, "", err
			}
			if manifestDescriptorElem.Platform != nil {
				if manifestDescriptorElem.Platform.OS != i.sysctx.OSChoice || manifestDescriptorElem.Platform.Architecture != i.sysctx.ArchitectureChoice {
					continue
				}
			}

			platformSpecManifest, singleManifestByte, singleManifestType, err := ManifestHandler(manifestByte, manifestType, i)
			if err != nil {
				return nil, nil, "", err
			}
			realManifestByte = singleManifestByte
			realManifestType = singleManifestType
			manifestInfoSlice = append(manifestInfoSlice, platformSpecManifest...)
		}
		return manifestInfoSlice, realManifestByte, realManifestType, nil
	}
	return nil, nil, "", fmt.Errorf("unsupported manifest type: %v", t)
}
