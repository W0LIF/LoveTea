package planning

import (
	"net/http/httptest"
	"testing"
)

func TestParseListPage(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    ListPage
		wantErr bool
	}{
		{name: "defaults", want: ListPage{Limit: defaultListLimit}},
		{name: "explicit page", query: "?limit=25&offset=50", want: ListPage{Limit: 25, Offset: 50}},
		{name: "zero limit", query: "?limit=0", wantErr: true},
		{name: "too large limit", query: "?limit=201", wantErr: true},
		{name: "negative offset", query: "?offset=-1", wantErr: true},
		{name: "too large offset", query: "?offset=1000001", wantErr: true},
		{name: "duplicate limit", query: "?limit=10&limit=20", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/tasks"+test.query, nil)
			got, err := parseListPage(request)
			if (err != nil) != test.wantErr {
				t.Fatalf("parseListPage() error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Fatalf("parseListPage() = %+v, want %+v", got, test.want)
			}
		})
	}
}
