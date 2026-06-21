package core

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestNewOCILayout(t *testing.T) {
	tmpDir := t.TempDir()
	layoutDir := filepath.Join(tmpDir, "oci-layout-test")

	oli, err := NewOCILayout(layoutDir)
	if err != nil {
		t.Fatalf("NewOCILayout failed: %v", err)
	}
	if oli == nil {
		t.Fatal("NewOCILayout returned nil")
	}

	// Verify directory structure
	if _, err := os.Stat(filepath.Join(layoutDir, "oci-layout")); err != nil {
		t.Errorf("oci-layout file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(layoutDir, "blobs", "sha256")); err != nil {
		t.Errorf("blobs/sha256 directory missing: %v", err)
	}
}

func TestOCILayoutWriteAndReadBlob(t *testing.T) {
	tmpDir := t.TempDir()
	oli, err := NewOCILayout(filepath.Join(tmpDir, "test"))
	if err != nil {
		t.Fatalf("NewOCILayout failed: %v", err)
	}

	content := []byte("hello oci layout")
	d := digest.FromBytes(content)

	// Write blob
	if err := oli.WriteBlob(d, bytes.NewReader(content), int64(len(content))); err != nil {
		t.Fatalf("WriteBlob failed: %v", err)
	}

	// Verify blob exists
	if !oli.BlobExists(d) {
		t.Fatal("BlobExists returned false after write")
	}

	// Read blob back
	rdr, err := oli.OpenBlob(d)
	if err != nil {
		t.Fatalf("OpenBlob failed: %v", err)
	}
	defer rdr.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(rdr); err != nil {
		t.Fatalf("Read blob failed: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), content) {
		t.Fatalf("Blob content mismatch: got %q, want %q", buf.Bytes(), content)
	}
}

func TestOCILayoutWriteBlobIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	oli, err := NewOCILayout(filepath.Join(tmpDir, "test"))
	if err != nil {
		t.Fatalf("NewOCILayout failed: %v", err)
	}

	content := []byte("idempotent test")
	d := digest.FromBytes(content)

	if err := oli.WriteBlob(d, bytes.NewReader(content), int64(len(content))); err != nil {
		t.Fatalf("First WriteBlob failed: %v", err)
	}

	// Second write should succeed without error
	if err := oli.WriteBlob(d, bytes.NewReader(content), int64(len(content))); err != nil {
		t.Fatalf("Second WriteBlob failed: %v", err)
	}
}

func TestOCILayoutIndex(t *testing.T) {
	tmpDir := t.TempDir()
	oli, err := NewOCILayout(filepath.Join(tmpDir, "test"))
	if err != nil {
		t.Fatalf("NewOCILayout failed: %v", err)
	}

	manifest := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`)
	d := digest.FromBytes(manifest)

	if err := oli.WriteBlob(d, bytes.NewReader(manifest), int64(len(manifest))); err != nil {
		t.Fatalf("WriteBlob failed: %v", err)
	}

	desc := imgspecv1.Descriptor{
		MediaType: imgspecv1.MediaTypeImageManifest,
		Digest:    d,
		Size:      int64(len(manifest)),
		Annotations: map[string]string{
			imgspecv1.AnnotationRefName: "test:latest",
		},
	}
	if err := oli.AddManifest(desc); err != nil {
		t.Fatalf("AddManifest failed: %v", err)
	}

	if err := oli.SaveIndex(); err != nil {
		t.Fatalf("SaveIndex failed: %v", err)
	}

	// Load index in a new instance
	oli2, err := NewOCILayout(filepath.Join(tmpDir, "test"))
	if err != nil {
		t.Fatalf("NewOCILayout (reload) failed: %v", err)
	}

	foundDesc, foundManifest, err := oli2.FindManifestByRef("test:latest")
	if err != nil {
		t.Fatalf("FindManifestByRef failed: %v", err)
	}
	if foundDesc.Digest != d {
		t.Fatalf("Manifest digest mismatch: got %v, want %v", foundDesc.Digest, d)
	}
	if !bytes.Equal(foundManifest, manifest) {
		t.Fatalf("Manifest content mismatch")
	}
}

func TestOCILayoutListRefs(t *testing.T) {
	tmpDir := t.TempDir()
	oli, err := NewOCILayout(filepath.Join(tmpDir, "test"))
	if err != nil {
		t.Fatalf("NewOCILayout failed: %v", err)
	}

	for _, ref := range []string{"a:v1", "b:v2", "c:v3"} {
		manifest := []byte(`{"schemaVersion":2}`)
		d := digest.FromBytes(manifest)
		oli.WriteBlob(d, bytes.NewReader(manifest), int64(len(manifest)))
		oli.AddManifest(imgspecv1.Descriptor{
			Digest: d,
			Annotations: map[string]string{
				imgspecv1.AnnotationRefName: ref,
			},
		})
	}

	refs := oli.ListRefs()
	if len(refs) != 3 {
		t.Fatalf("Expected 3 refs, got %d", len(refs))
	}
	for i, expected := range []string{"a:v1", "b:v2", "c:v3"} {
		if refs[i] != expected {
			t.Errorf("Ref[%d] = %q, want %q", i, refs[i], expected)
		}
	}
}

func TestOCILayoutListBlobs(t *testing.T) {
	tmpDir := t.TempDir()
	oli, err := NewOCILayout(filepath.Join(tmpDir, "test"))
	if err != nil {
		t.Fatalf("NewOCILayout failed: %v", err)
	}

	contents := [][]byte{
		[]byte("blob1"),
		[]byte("blob2"),
	}
	for _, c := range contents {
		d := digest.FromBytes(c)
		if err := oli.WriteBlob(d, bytes.NewReader(c), int64(len(c))); err != nil {
			t.Fatalf("WriteBlob failed: %v", err)
		}
	}

	blobs, err := oli.ListBlobs()
	if err != nil {
		t.Fatalf("ListBlobs failed: %v", err)
	}
	if len(blobs) != 2 {
		t.Fatalf("Expected 2 blobs, got %d", len(blobs))
	}
}

func TestOCILayoutSizeMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	oli, err := NewOCILayout(filepath.Join(tmpDir, "test"))
	if err != nil {
		t.Fatalf("NewOCILayout failed: %v", err)
	}

	content := []byte("short")
	d := digest.FromBytes(content)

	if err := oli.WriteBlob(d, bytes.NewReader(content), 100); err == nil {
		t.Fatal("Expected size mismatch error, got nil")
	} else if !strings.Contains(err.Error(), "size mismatch") {
		t.Fatalf("Unexpected error message: %v", err)
	}
}
