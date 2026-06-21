package core

import (
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"testing"

	"github.com/containers/image/v5/types"
)

func TestReaderSumWrapper(t *testing.T) {
	content := []byte("hello world")
	r := NewReaderSumWrapper(bytes.NewReader(content))

	buf := make([]byte, 5)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 5 {
		t.Fatalf("Read returned %d bytes, expected 5", n)
	}
	if r.Size != 5 {
		t.Fatalf("Size = %d after first read, expected 5", r.Size)
	}

	// Read remaining
	remaining, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if !bytes.Equal(remaining, []byte(" world")) {
		t.Fatalf("Remaining content mismatch: got %q", remaining)
	}
	if r.Size != int64(len(content)) {
		t.Fatalf("Final Size = %d, expected %d", r.Size, len(content))
	}
}

func TestManifestJSON(t *testing.T) {
	m := Manifest{
		Config: types.BlobInfo{Digest: "sha256:abc123", Size: 100},
		Layers: []types.BlobInfo{
			{Digest: "sha256:layer1", Size: 200},
			{Digest: "sha256:layer2", Size: 300},
		},
	}

	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Manifest
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Config.Digest.String() != "sha256:abc123" {
		t.Errorf("Config digest mismatch: got %v", decoded.Config.Digest)
	}
	if len(decoded.Layers) != 2 {
		t.Fatalf("Expected 2 layers, got %d", len(decoded.Layers))
	}
	if decoded.Layers[0].Size != 200 {
		t.Errorf("Layer[0] size mismatch: got %d", decoded.Layers[0].Size)
	}
}

func TestGenRepoUrl(t *testing.T) {
	tests := []struct {
		srcReg  string
		dstReg  string
		dstRepo string
		rawURL  string
		wantSrc string
		wantDst string
	}{
		{
			srcReg:  "hub.docker.com",
			dstReg:  "harbor.example.com",
			dstRepo: "",
			rawURL:  "library/nginx:1.20",
			wantSrc: "hub.docker.com/library/nginx:1.20",
			wantDst: "harbor.example.com/library/nginx:1.20",
		},
		{
			srcReg:  "hub.docker.com",
			dstReg:  "harbor.example.com",
			dstRepo: "proj",
			rawURL:  "library/nginx:1.20",
			wantSrc: "hub.docker.com/library/nginx:1.20",
			wantDst: "harbor.example.com/proj/nginx:1.20",
		},
		{
			srcReg:  "",
			dstReg:  "harbor.example.com",
			dstRepo: "",
			rawURL:  "library/nginx:1.20",
			wantSrc: "library/nginx:1.20",
			wantDst: "harbor.example.com/library/nginx:1.20",
		},
	}

	for _, tt := range tests {
		gotSrc, gotDst := GenRepoUrl(tt.srcReg, tt.dstReg, tt.dstRepo, tt.rawURL)
		if gotSrc != tt.wantSrc {
			t.Errorf("GenRepoUrl(src=%q, dst=%q, repo=%q, raw=%q) src = %q, want %q",
				tt.srcReg, tt.dstReg, tt.dstRepo, tt.rawURL, gotSrc, tt.wantSrc)
		}
		if gotDst != tt.wantDst {
			t.Errorf("GenRepoUrl(src=%q, dst=%q, repo=%q, raw=%q) dst = %q, want %q",
				tt.srcReg, tt.dstReg, tt.dstRepo, tt.rawURL, gotDst, tt.wantDst)
		}
	}
}

func TestCheckInvalidChar(t *testing.T) {
	if CheckInvalidChar("hello") {
		t.Error("CheckInvalidChar(\"hello\") should be false")
	}
	if !CheckInvalidChar("hello\x00") {
		t.Error("CheckInvalidChar with null byte should be true")
	}
	if !CheckInvalidChar("hello\x1f") {
		t.Error("CheckInvalidChar with control char should be true")
	}
}

func TestFormatByteSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0.0B"},
		{512, "512.0B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1024 * 1024, "1.0MB"},
		{1024 * 1024 * 1024, "1.0GB"},
	}

	for _, tt := range tests {
		got := FormatByteSize(tt.bytes)
		if got != tt.want {
			t.Errorf("FormatByteSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestShortenString(t *testing.T) {
	if got := ShortenString("hello", 3); got != "hel" {
		t.Errorf("ShortenString(\"hello\", 3) = %q, want %q", got, "hel")
	}
	if got := ShortenString("hi", 5); got != "hi" {
		t.Errorf("ShortenString(\"hi\", 5) = %q, want %q", got, "hi")
	}
}

func TestGetBlobSuffix(t *testing.T) {
	tests := []struct {
		mediaType string
		size      int64
		want      string
	}{
		{"application/vnd.docker.image.rootfs.diff.tar.gzip", 100, ".tar.gz"},
		{"application/vnd.oci.image.layer.v1.tar+gzip", 100, ".tar.gz"},
		{"application/vnd.oci.image.layer.v1.tar+zstd", 100, ".tar.zst"},
		{"application/vnd.docker.image.rootfs.diff.tar.gzip", 10, ".raw"},
		{"application/vnd.docker.image.rootfs.diff.tar", 100, ".tar"},
		{"application/octet-stream", 100, ".raw"},
	}

	for _, tt := range tests {
		bi := types.BlobInfo{MediaType: tt.mediaType, Size: tt.size}
		got := GetBlobSuffix(bi)
		if got != tt.want {
			t.Errorf("GetBlobSuffix(%q, size=%d) = %q, want %q", tt.mediaType, tt.size, got, tt.want)
		}
	}
}

func TestTaskContextReset(t *testing.T) {
	ctx := NewTaskContext(NewCmdLogger(), nil, nil)
	ctx.Reset()

	if ctx.Cancel() {
		t.Error("Context should not be cancelled after Reset")
	}
	if ctx.OCILayout != nil {
		t.Error("OCILayout should be nil after Reset")
	}
	if ctx.StoreFormat != "" {
		t.Errorf("StoreFormat should be empty after Reset, got %q", ctx.StoreFormat)
	}
}

func TestTaskContextCreateOCILayout(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := NewTaskContext(NewCmdLogger(), nil, nil)
	ctx.Reset()

	ociPath := filepath.Join(tmpDir, "oci")
	if err := ctx.CreateOCILayout(ociPath); err != nil {
		t.Fatalf("CreateOCILayout failed: %v", err)
	}
	if ctx.OCILayout == nil {
		t.Fatal("OCILayout should not be nil after CreateOCILayout")
	}
	if ctx.StoreFormat != "oci" {
		t.Errorf("StoreFormat = %q, want oci", ctx.StoreFormat)
	}
}
