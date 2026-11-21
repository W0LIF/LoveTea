package storage

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/W0LIF/LoveTea/backend/internal/config"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func TestMinIOPresignedUploadAndDownload(t *testing.T) {
	if os.Getenv("LOVETEA_MINIO_INTEGRATION") != "1" {
		t.Skip("set LOVETEA_MINIO_INTEGRATION=1 to run against MinIO")
	}

	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("find repository root: %v", err)
	}
	if err := godotenv.Load(filepath.Join(root, ".env")); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("load local environment: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	store, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("create MinIO client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := store.EnsureBucket(ctx); err != nil {
		t.Fatalf("ensure bucket: %v", err)
	}

	objectKey := "integration/" + uuid.NewString()
	form, err := store.CreateUploadForm(ctx, objectKey, "image/png")
	if err != nil {
		t.Fatalf("create upload form: %v", err)
	}
	if form.URL == "" || form.Fields["key"] != objectKey {
		t.Fatal("upload form is missing its URL or object key")
	}

	testImage := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	testImage.SetNRGBA(0, 0, color.NRGBA{R: 32, G: 64, B: 128, A: 255})
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, testImage); err != nil {
		t.Fatalf("encode test image: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range form.Fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("write upload form field: %v", err)
		}
	}
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", `form-data; name="file"; filename="upload.png"`)
	partHeader.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("create file form part: %v", err)
	}
	if _, err := part.Write(pngData.Bytes()); err != nil {
		t.Fatalf("write test image: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("finish upload form: %v", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, form.URL, &body)
	if err != nil {
		t.Fatalf("create upload request: %v", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post object using signed form: %v", err)
	}
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 1024))
	_ = response.Body.Close()
	if readErr != nil {
		t.Fatalf("read upload response: %v", readErr)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		t.Fatalf("upload status = %d, want a 2xx response: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	uploaded := true
	defer func() {
		if uploaded {
			if err := store.Delete(context.Background(), objectKey); err != nil {
				t.Errorf("clean up uploaded object: %v", err)
			}
		}
	}()

	info, err := store.Inspect(ctx, objectKey)
	if err != nil {
		t.Fatalf("inspect uploaded object: %v", err)
	}
	if info.ContentType != "image/png" || info.Format != "png" || info.Size != int64(pngData.Len()) {
		t.Fatalf("inspected object has unexpected metadata: %+v", info)
	}

	downloadURL, expiresAt, err := store.CreateDownloadURL(ctx, objectKey)
	if err != nil {
		t.Fatalf("create download URL: %v", err)
	}
	if downloadURL == "" || !expiresAt.After(time.Now()) {
		t.Fatal("download URL is missing or already expired")
	}
	downloadRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		t.Fatalf("create download request: %v", err)
	}
	downloadResponse, err := http.DefaultClient.Do(downloadRequest)
	if err != nil {
		t.Fatalf("download object using signed URL: %v", err)
	}
	downloaded, readErr := io.ReadAll(downloadResponse.Body)
	_ = downloadResponse.Body.Close()
	if readErr != nil {
		t.Fatalf("read downloaded object: %v", readErr)
	}
	if downloadResponse.StatusCode != http.StatusOK || !bytes.Equal(downloaded, pngData.Bytes()) {
		t.Fatalf("download response status = %d or object contents differ", downloadResponse.StatusCode)
	}

	if err := store.Delete(ctx, objectKey); err != nil {
		t.Fatalf("delete uploaded object: %v", err)
	}
	uploaded = false
}
