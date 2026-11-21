package content

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/W0LIF/LoveTea/backend/internal/platform/storage"
	"github.com/google/uuid"
)

type fakeRepository struct {
	media        MediaFile
	objectKey    string
	createdScope string
	createdType  string
	createdID    uuid.UUID
	completed    int
	moment       Moment
	err          error
}

func (r *fakeRepository) ListMedia(context.Context, uuid.UUID) ([]MediaFile, error) {
	return []MediaFile{}, nil
}

func (r *fakeRepository) CreatePendingMedia(
	_ context.Context,
	_ uuid.UUID,
	id uuid.UUID,
	key string,
	contentType string,
	scope string,
) (MediaFile, error) {
	r.createdID, r.objectKey, r.createdType, r.createdScope = id, key, contentType, scope
	return MediaFile{ID: id.String(), Scope: scope, ContentType: contentType, Status: "pending"}, r.err
}

func (r *fakeRepository) GetMedia(
	context.Context,
	uuid.UUID,
	uuid.UUID,
) (MediaFile, string, error) {
	return r.media, r.objectKey, r.err
}

func (r *fakeRepository) CompleteMedia(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
	size int64,
) (MediaFile, string, error) {
	r.completed++
	r.media.Status = "ready"
	r.media.SizeBytes = size
	return r.media, r.objectKey, r.err
}

func (r *fakeRepository) DeleteMedia(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	int64,
) (string, error) {
	return r.objectKey, r.err
}

func (r *fakeRepository) ListMoments(context.Context, uuid.UUID) ([]Moment, error) {
	return []Moment{}, nil
}

func (r *fakeRepository) CreateMoment(
	_ context.Context,
	_ uuid.UUID,
	request CreateMomentRequest,
	mediaIDs []uuid.UUID,
) (Moment, error) {
	r.moment = Moment{Title: request.Title, Caption: request.Caption}
	return r.moment, r.err
}

type fakeObjectStore struct {
	form     storage.UploadForm
	info     storage.ObjectInfo
	download string
	deleted  string
	err      error
}

func (s *fakeObjectStore) CreateUploadForm(
	_ context.Context,
	_ string,
	_ string,
) (storage.UploadForm, error) {
	return s.form, s.err
}

func (s *fakeObjectStore) CreateDownloadURL(
	context.Context,
	string,
) (string, time.Time, error) {
	return s.download, time.Now().Add(time.Minute), s.err
}

func (s *fakeObjectStore) Inspect(context.Context, string) (storage.ObjectInfo, error) {
	return s.info, s.err
}

func (s *fakeObjectStore) Delete(_ context.Context, key string) error {
	s.deleted = key
	return s.err
}

func TestCreateUploadRestrictsTypeAndReturnsPolicy(t *testing.T) {
	repo := &fakeRepository{}
	store := &fakeObjectStore{form: storage.UploadForm{
		URL:       "https://storage.example/upload",
		Fields:    map[string]string{"policy": "signed"},
		ExpiresAt: time.Now().Add(time.Minute),
	}}
	service := NewService(repo, store)
	userID := uuid.New()

	if _, err := service.CreateUpload(context.Background(), userID, UploadRequest{
		Scope: "personal", ContentType: "application/octet-stream",
	}); !errors.Is(err, ErrInvalidContent) {
		t.Fatalf("CreateUpload() error = %v, want invalid content type", err)
	}

	result, err := service.CreateUpload(context.Background(), userID, UploadRequest{
		Scope: "personal", ContentType: "image/png",
	})
	if err != nil {
		t.Fatalf("CreateUpload() error = %v", err)
	}
	if result.Media.Status != "pending" || result.UploadURL != store.form.URL ||
		repo.createdType != "image/png" || repo.createdScope != "personal" {
		t.Fatalf("CreateUpload() returned unexpected result: %+v", result)
	}
}

func TestCompleteUploadValidatesStoredObjectBeforePublishing(t *testing.T) {
	repo := &fakeRepository{
		media:     MediaFile{ID: uuid.NewString(), ContentType: "image/png", Status: "pending"},
		objectKey: "media/test",
	}
	store := &fakeObjectStore{info: storage.ObjectInfo{Size: 10, ContentType: "image/jpeg", Format: "jpeg"}}
	service := NewService(repo, store)

	if _, err := service.CompleteUpload(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, ErrInvalidContent) {
		t.Fatalf("CompleteUpload() error = %v, want content type mismatch", err)
	}
	if repo.completed != 0 {
		t.Fatal("CompleteUpload() published object with a mismatched content type")
	}

	store.info.ContentType = "image/png"
	store.info.Format = "png"
	media, err := service.CompleteUpload(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("CompleteUpload() error = %v", err)
	}
	if media.Status != "ready" || media.SizeBytes != 10 || repo.completed != 1 {
		t.Fatalf("CompleteUpload() did not publish verified object: %+v", media)
	}
}

func TestCreateMomentRejectsDuplicateMediaIDs(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, &fakeObjectStore{})
	id := uuid.NewString()

	_, err := service.CreateMoment(context.Background(), uuid.New(), CreateMomentRequest{
		Scope:      "personal",
		Title:      "Walk",
		HappenedAt: time.Now(),
		MediaIDs:   []string{id, id},
	})
	if !errors.Is(err, ErrInvalidContent) {
		t.Fatalf("CreateMoment() error = %v, want duplicate media rejection", err)
	}
	if repo.moment.Title != "" {
		t.Fatal("CreateMoment() persisted a request with duplicate media IDs")
	}
}
