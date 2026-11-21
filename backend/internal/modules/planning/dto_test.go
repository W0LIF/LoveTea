package planning

import (
	"encoding/json"
	"testing"
)

func TestUpdateTaskRequestDistinguishesOmittedAndClearedValues(t *testing.T) {
	var request UpdateTaskRequest
	if err := json.Unmarshal([]byte(`{"version":3,"due_at":null,"color":null}`), &request); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !request.DueAt.Set || request.DueAt.Value != nil {
		t.Fatalf("due_at should be explicitly cleared: %+v", request.DueAt)
	}
	if !request.Color.Set || request.Color.Value != nil {
		t.Fatalf("color should be explicitly cleared: %+v", request.Color)
	}
	if request.ReminderAt.Set {
		t.Fatal("omitted reminder_at must remain unchanged")
	}
}
