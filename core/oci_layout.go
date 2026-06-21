package core

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// OCILayout manages a standard OCI Image Layout directory.
type OCILayout struct {
	rootPath string
	index    imgspecv1.Index
	mu       sync.Mutex // protects in-memory index only
}

// NewOCILayout creates or opens an OCI Image Layout at rootPath.
func NewOCILayout(rootPath string) (*OCILayout, error) {
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		return nil, fmt.Errorf("create oci layout root: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(rootPath, "blobs", "sha256"), 0755); err != nil {
		return nil, fmt.Errorf("create blobs/sha256: %w", err)
	}

	layoutFile := filepath.Join(rootPath, "oci-layout")
	if _, err := os.Stat(layoutFile); os.IsNotExist(err) {
		layout := map[string]string{"imageLayoutVersion": "1.0.0"}
		b, _ := json.Marshal(layout)
		if err := os.WriteFile(layoutFile, b, 0644); err != nil {
			return nil, fmt.Errorf("write oci-layout: %w", err)
		}
	}

	oli := &OCILayout{
		rootPath: rootPath,
		index: imgspecv1.Index{
			Versioned: specs.Versioned{
				SchemaVersion: 2,
			},
		},
	}
	// Try to load existing index.json
	_ = oli.LoadIndex()
	return oli, nil
}

// blobPath returns the filesystem path for a given digest.
func (o *OCILayout) blobPath(d digest.Digest) string {
	return filepath.Join(o.rootPath, "blobs", d.Algorithm().String(), d.Hex())
}

// WriteBlob writes a blob to the content-addressable store. Idempotent.
// Uses atomic temp-file + rename so multiple goroutines can write different blobs concurrently.
func (o *OCILayout) WriteBlob(d digest.Digest, r io.Reader, size int64) error {
	path := o.blobPath(d)

	// Fast path: blob already exists, drain reader and return.
	if _, err := os.Stat(path); err == nil {
		_, _ = io.Copy(io.Discard, r)
		return nil
	}

	tmpPath := path + ".tmp"

	// Try to create the temp file exclusively (O_EXCL fails if another goroutine already created it).
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			// Another goroutine is writing this blob concurrently — drain the reader.
			_, _ = io.Copy(io.Discard, r)
			// Wait for the other goroutine to finish (rename tmp → final).
			for {
				time.Sleep(50 * time.Millisecond)
				if _, statErr := os.Stat(path); statErr == nil {
					return nil
				}
				// If tmp file disappeared, the writer failed — fall through to retry.
				if _, statErr := os.Stat(tmpPath); os.IsNotExist(statErr) {
					break
				}
			}
			// Retry: the other writer failed, so we try again.
			return o.WriteBlob(d, r, size)
		}
		// Race between Stat and OpenFile — check final path again.
		if _, statErr := os.Stat(path); statErr == nil {
			_, _ = io.Copy(io.Discard, r)
			return nil
		}
		return fmt.Errorf("create blob temp file: %w", err)
	}

	written, err := io.Copy(f, r)
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("write blob: %w", err)
	}
	if size > 0 && written != size {
		os.Remove(tmpPath)
		return fmt.Errorf("blob size mismatch: wrote %d, expected %d", written, size)
	}

	// Atomic rename — if another goroutine already renamed, that's fine (same content).
	if err := os.Rename(tmpPath, path); err != nil {
		if _, statErr := os.Stat(path); statErr == nil {
			os.Remove(tmpPath)
			return nil
		}
		os.Remove(tmpPath)
		return fmt.Errorf("rename blob: %w", err)
	}
	return nil
}

// BlobExists reports whether a blob already exists in the store.
func (o *OCILayout) BlobExists(d digest.Digest) bool {
	_, err := os.Stat(o.blobPath(d))
	return err == nil
}

// OpenBlob opens an existing blob for reading.
func (o *OCILayout) OpenBlob(d digest.Digest) (io.ReadCloser, error) {
	f, err := os.Open(o.blobPath(d))
	if err != nil {
		return nil, fmt.Errorf("open blob %s: %w", d, err)
	}
	return f, nil
}

// AddManifest appends a manifest descriptor to the index.
// If a manifest with the same ref name already exists, it is replaced.
func (o *OCILayout) AddManifest(desc imgspecv1.Descriptor) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	refName := desc.Annotations[imgspecv1.AnnotationRefName]
	if refName != "" {
		for i, existing := range o.index.Manifests {
			if existing.Annotations[imgspecv1.AnnotationRefName] == refName {
				o.index.Manifests[i] = desc
				return nil
			}
		}
	}
	o.index.Manifests = append(o.index.Manifests, desc)
	return nil
}

// SaveIndex writes the current index to index.json.
func (o *OCILayout) SaveIndex() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	b, err := json.MarshalIndent(o.index, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	path := filepath.Join(o.rootPath, "index.json")
	if err := os.WriteFile(path, b, 0644); err != nil {
		return fmt.Errorf("write index.json: %w", err)
	}
	return nil
}

// LoadIndex reads index.json from disk.
func (o *OCILayout) LoadIndex() error {
	path := filepath.Join(o.rootPath, "index.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read index.json: %w", err)
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	return json.Unmarshal(b, &o.index)
}

// FindManifestByRef looks up a manifest descriptor by its org.opencontainers.image.ref.name annotation.
func (o *OCILayout) FindManifestByRef(imageRef string) (imgspecv1.Descriptor, []byte, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, desc := range o.index.Manifests {
		if desc.Annotations[imgspecv1.AnnotationRefName] == imageRef {
			b, err := os.ReadFile(o.blobPath(desc.Digest))
			if err != nil {
				return imgspecv1.Descriptor{}, nil, fmt.Errorf("read manifest blob: %w", err)
			}
			return desc, b, nil
		}
	}
	return imgspecv1.Descriptor{}, nil, fmt.Errorf("manifest for %s not found", imageRef)
}

// ListRefs returns all image ref names stored in the OCI index.
func (o *OCILayout) ListRefs() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	var refs []string
	for _, desc := range o.index.Manifests {
		if ref, ok := desc.Annotations[imgspecv1.AnnotationRefName]; ok {
			refs = append(refs, ref)
		}
	}
	return refs
}

// ListBlobs returns all blob digests present in blobs/sha256/.
func (o *OCILayout) ListBlobs() ([]digest.Digest, error) {
	dir := filepath.Join(o.rootPath, "blobs", "sha256")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read blobs dir: %w", err)
	}

	var digests []digest.Digest
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.Contains(name, ".") {
			continue
		}
		d, err := digest.Parse("sha256:" + name)
		if err != nil {
			continue
		}
		digests = append(digests, d)
	}
	return digests, nil
}

func (o *OCILayout) PackToTar(tarPath string) error {
	f, err := os.Create(tarPath)
	if err != nil {
		return fmt.Errorf("create tar file: %w", err)
	}
	defer f.Close()

	var wc io.WriteCloser = f
	if strings.HasSuffix(tarPath, ".tar.gz") || strings.HasSuffix(tarPath, ".tgz") {
		wc = gzip.NewWriter(f)
		defer wc.Close()
	}

	tw := tar.NewWriter(wc)
	defer tw.Close()

	err = filepath.Walk(o.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(o.rootPath, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		hdr := &tar.Header{
			Name: rel,
			Size: info.Size(),
			Mode: int64(info.Mode()),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("write tar header for %s: %w", rel, err)
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open file %s: %w", path, err)
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return fmt.Errorf("write file %s to tar: %w", rel, err)
		}
		f.Close()
		return nil
	})
	return err
}

func ExtractFromTar(tarPath string, destDir string) (*OCILayout, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return nil, fmt.Errorf("open tar file: %w", err)
	}
	defer f.Close()

	var tr *tar.Reader
	if strings.HasSuffix(tarPath, ".tar.gz") || strings.HasSuffix(tarPath, ".tgz") {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		defer gzr.Close()
		tr = tar.NewReader(gzr)
	} else {
		tr = tar.NewReader(f)
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar entry: %w", err)
		}

		name := hdr.Name
		name = strings.ReplaceAll(name, "\\", "/")

		if strings.HasPrefix(name, "blobs/") || strings.HasPrefix(name, "oci-layout") || strings.HasPrefix(name, "index.json") {
			// use as-is
		} else {
			slashIdx := strings.Index(name, "/")
			if slashIdx >= 0 {
				name = name[slashIdx+1:]
			}
		}

		if name == "" {
			continue
		}

		targetPath := filepath.Join(destDir, name)
		dir := filepath.Dir(targetPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create dir %s: %w", dir, err)
		}

		outf, err := os.Create(targetPath)
		if err != nil {
			return nil, fmt.Errorf("create file %s: %w", targetPath, err)
		}
		if _, err := io.Copy(outf, tr); err != nil {
			outf.Close()
			return nil, fmt.Errorf("write file %s: %w", targetPath, err)
		}
		outf.Close()
	}

	layoutFile := filepath.Join(destDir, "oci-layout")
	if _, err := os.Stat(layoutFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("not an OCI layout tar: oci-layout file not found")
	}

	return NewOCILayout(destDir)
}

func (o *OCILayout) CopyBlobsFrom(src *OCILayout) error {
	blobs, err := src.ListBlobs()
	if err != nil {
		return fmt.Errorf("list blobs from source: %w", err)
	}

	for _, d := range blobs {
		if o.BlobExists(d) {
			continue
		}

		rdr, err := src.OpenBlob(d)
		if err != nil {
			return fmt.Errorf("open blob %s from source: %w", d, err)
		}

		path := src.blobPath(d)
		fi, err := os.Stat(path)
		var size int64
		if err == nil {
			size = fi.Size()
		}

		if err := o.WriteBlob(d, rdr, size); err != nil {
			rdr.Close()
			return fmt.Errorf("copy blob %s: %w", d, err)
		}
		rdr.Close()
	}
	return nil
}

func IsOCILayoutTar(tarPath string) bool {
	f, err := os.Open(tarPath)
	if err != nil {
		return false
	}
	defer f.Close()

	var tr *tar.Reader
	if strings.HasSuffix(tarPath, ".tar.gz") || strings.HasSuffix(tarPath, ".tgz") {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			return false
		}
		defer gzr.Close()
		tr = tar.NewReader(gzr)
	} else {
		tr = tar.NewReader(f)
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false
		}
		name := hdr.Name
		name = strings.ReplaceAll(name, "\\", "/")
		if filepath.Base(name) == "oci-layout" {
			return true
		}
	}
	return false
}

// DockerManifestEntry represents a single image entry in Docker's manifest.json
type DockerManifestEntry struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

// PackToDockerTar generates a tar that is simultaneously valid as both an OCI Image Layout
// and a Docker V2 image tar. When dockerOnly is true, OCI layout files are omitted.
// The tar contains:
//   - OCI layout files (unless dockerOnly): oci-layout, index.json, blobs/sha256/<hex>
//   - Docker files: manifest.json, <hex>.json (config), <hex>/layer.tar (hardlink to blob)
//
// Layer data is stored only once in blobs/sha256/. Docker layer.tar entries are hardlinks
// to avoid data duplication. If localStore is provided, missing blobs will be read from it.
// In incremental mode, missing blobs are skipped but the Docker manifest still references
// all layers so the loading tool can resolve them from the target host.
func (o *OCILayout) PackToDockerTar(tarPath string, localStore *OCILayout, dockerOnly bool) error {
	f, err := os.Create(tarPath)
	if err != nil {
		return fmt.Errorf("create tar file: %w", err)
	}
	defer f.Close()

	tw := tar.NewWriter(f)
	defer tw.Close()

	// Phase 1: Write OCI layout files (skip if docker-only)
	if !dockerOnly {
		if err := writeTarEntry(tw, "oci-layout", []byte(`{"imageLayoutVersion":"1.0.0"}`)); err != nil {
			return fmt.Errorf("write oci-layout: %w", err)
		}
	}

	// Phase 2: Collect all blobs needed and write them to blobs/sha256/
	var dockerManifest []DockerManifestEntry
	writtenBlobs := make(map[string]bool) // hex -> already written in blobs/
	blobSizes := make(map[string]int64)   // hex -> file size (for hardlinks)

	type layerInfo struct {
		digest digest.Digest
		hex    string
	}
	type imageInfo struct {
		manifestDesc imgspecv1.Descriptor
		configDigest digest.Digest
		configHex    string
		layers       []layerInfo
	}

	var images []imageInfo
	allBlobs := make(map[string]digest.Digest) // hex -> digest

	for _, desc := range o.index.Manifests {
		manifestBytes, err := readBlobFromLayout(o, localStore, desc.Digest)
		if err != nil {
			return fmt.Errorf("read manifest blob %s: %w", desc.Digest, err)
		}

		var ociManifest struct {
			Config struct {
				Digest digest.Digest `json:"digest"`
			} `json:"config"`
			Layers []struct {
				Digest digest.Digest `json:"digest"`
			} `json:"layers"`
		}
		if err := json.Unmarshal(manifestBytes, &ociManifest); err != nil {
			return fmt.Errorf("parse manifest: %w", err)
		}

		img := imageInfo{
			manifestDesc: desc,
			configDigest: ociManifest.Config.Digest,
			configHex:    ociManifest.Config.Digest.Hex(),
		}

		allBlobs[desc.Digest.Hex()] = desc.Digest
		allBlobs[img.configHex] = img.configDigest

		for _, layer := range ociManifest.Layers {
			hex := layer.Digest.Hex()
			img.layers = append(img.layers, layerInfo{digest: layer.Digest, hex: hex})
			allBlobs[hex] = layer.Digest
		}

		images = append(images, img)
	}

	// Write all available blobs to blobs/sha256/ (skip missing ones for incremental mode)
	if !dockerOnly {
		for hex, d := range allBlobs {
			if writtenBlobs[hex] {
				continue
			}

			path := o.blobPath(d)
			fi, statErr := os.Stat(path)
			if statErr != nil && localStore != nil {
				path = localStore.blobPath(d)
				fi, statErr = os.Stat(path)
			}
			if statErr != nil {
				continue
			}
			blobSizes[hex] = fi.Size()

			tarName := "blobs/sha256/" + hex
			rf, err := os.Open(path)
			if err != nil {
				continue
			}

			hdr := &tar.Header{
				Name: tarName,
				Size: fi.Size(),
				Mode: 0644,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				rf.Close()
				continue
			}
			if _, err := io.Copy(tw, rf); err != nil {
				rf.Close()
				continue
			}
			rf.Close()
			writtenBlobs[hex] = true
		}
	} else {
		// Docker-only: still need to track available blobs for size info
		for hex, d := range allBlobs {
			path := o.blobPath(d)
			fi, statErr := os.Stat(path)
			if statErr != nil && localStore != nil {
				path = localStore.blobPath(d)
				fi, statErr = os.Stat(path)
			}
			if statErr == nil {
				blobSizes[hex] = fi.Size()
				writtenBlobs[hex] = true
			}
		}
	}

	// Phase 3: Write Docker-specific files and build manifest.json
	writtenDockerEntries := make(map[string]bool)

	for _, img := range images {
		refName := img.manifestDesc.Annotations[imgspecv1.AnnotationRefName]
		if refName == "" {
			refName = "unknown:latest"
		}

		configFileName := img.configHex + ".json"

		// Write Docker config <hex>.json if available
		if writtenBlobs[img.configHex] && !writtenDockerEntries[configFileName] {
			var linkName string
			if dockerOnly {
				// Docker-only: write config as regular file
				configData, readErr := readBlobFromLayout(o, localStore, img.configDigest)
				if readErr == nil {
					writeTarEntry(tw, configFileName, configData)
				}
			} else {
				// Combined: write config as hardlink to blob
				hdr := &tar.Header{
					Name:     configFileName,
					Size:     blobSizes[img.configHex],
					Mode:     0644,
					Typeflag: tar.TypeLink,
					Linkname: "blobs/sha256/" + img.configHex,
				}
				if err := tw.WriteHeader(hdr); err == nil {
					linkName = configFileName
				}
			}
			_ = linkName
			writtenDockerEntries[configFileName] = true
		}

		// Write Docker layer directories (only for available blobs)
		var layerPaths []string
		for _, layer := range img.layers {
			layerDir := layer.hex
			layerPath := layerDir + "/layer.tar"

			// Always add to layer paths (manifest references all layers)
			layerPaths = append(layerPaths, layerPath)

			// Only write layer files if blob is available
			if !writtenBlobs[layer.hex] {
				continue
			}
			if writtenDockerEntries[layerDir] {
				continue
			}
			writtenDockerEntries[layerDir] = true

			if dockerOnly {
				// Docker-only: write layer data as regular file
				layerData, readErr := readBlobFromLayout(o, localStore, layer.digest)
				if readErr == nil {
					writeTarEntry(tw, layerDir+"/VERSION", []byte("1.0"))
					writeTarEntry(tw, layerDir+"/json", []byte(fmt.Sprintf(`{"id":"%s"}`, layer.hex)))
					writeTarEntry(tw, layerPath, layerData)
				}
			} else {
				// Combined: write layer metadata + hardlink
				writeTarEntry(tw, layerDir+"/VERSION", []byte("1.0"))
				writeTarEntry(tw, layerDir+"/json", []byte(fmt.Sprintf(`{"id":"%s"}`, layer.hex)))

				hdr := &tar.Header{
					Name:     layerPath,
					Size:     blobSizes[layer.hex],
					Mode:     0644,
					Typeflag: tar.TypeLink,
					Linkname: "blobs/sha256/" + layer.hex,
				}
				tw.WriteHeader(hdr)
			}
		}

		// Always include image in Docker manifest (even with missing layers)
		dockerManifest = append(dockerManifest, DockerManifestEntry{
			Config:   configFileName,
			RepoTags: []string{refName},
			Layers:   layerPaths,
		})
	}

	// Write manifest.json at tar root
	manifestJSON, err := json.Marshal(dockerManifest)
	if err != nil {
		return fmt.Errorf("marshal docker manifest: %w", err)
	}
	if err := writeTarEntry(tw, "manifest.json", manifestJSON); err != nil {
		return fmt.Errorf("write manifest.json: %w", err)
	}

	// Write OCI index.json (skip if docker-only)
	if !dockerOnly {
		indexJSON, err := json.MarshalIndent(o.index, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal index.json: %w", err)
		}
		return writeTarEntry(tw, "index.json", indexJSON)
	}

	return nil
}

// readBlobFromLayout reads a blob from the primary layout or fallback to local store
func readBlobFromLayout(primary *OCILayout, fallback *OCILayout, d digest.Digest) ([]byte, error) {
	data, err := os.ReadFile(primary.blobPath(d))
	if err == nil {
		return data, nil
	}
	if fallback != nil {
		data, err = os.ReadFile(fallback.blobPath(d))
		if err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("blob %s not found", d)
}

// streamBlobToTar reads a blob from primary or fallback layout and streams it to the tar writer.
// This avoids reading large layer blobs entirely into memory.
func streamBlobToTar(tw *tar.Writer, primary *OCILayout, fallback *OCILayout, d digest.Digest, tarName string) error {
	path := primary.blobPath(d)
	fi, err := os.Stat(path)
	if err != nil && fallback != nil {
		path = fallback.blobPath(d)
		fi, err = os.Stat(path)
	}
	if err != nil {
		return fmt.Errorf("blob %s not found", d)
	}

	rf, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open blob %s: %w", d, err)
	}
	defer rf.Close()

	hdr := &tar.Header{
		Name: tarName,
		Size: fi.Size(),
		Mode: 0644,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("write tar header for %s: %w", tarName, err)
	}
	if _, err := io.Copy(tw, rf); err != nil {
		return fmt.Errorf("write blob %s to tar: %w", tarName, err)
	}
	return nil
}

// writeTarEntry writes a single file entry to the tar writer
func writeTarEntry(tw *tar.Writer, name string, data []byte) error {
	hdr := &tar.Header{
		Name: name,
		Size: int64(len(data)),
		Mode: 0644,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("write tar header for %s: %w", name, err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("write tar data for %s: %w", name, err)
	}
	return nil
}
