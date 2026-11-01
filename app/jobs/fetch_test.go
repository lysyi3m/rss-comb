package jobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchURLRejectsOversizedBody(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{"at the limit", maxResponseBytes, false},
		{"over the limit", maxResponseBytes + 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write(make([]byte, tt.size))
			}))
			defer srv.Close()

			_, _, err := fetchURL(context.Background(), srv.URL, 30, srv.Client(), "test", false)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFetchURLReturnsFinalURLAfterRedirect(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/go", http.RedirectHandler("/posts/article/", http.StatusFound))
	mux.HandleFunc("/posts/article/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("page"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, finalURL, err := fetchURL(context.Background(), srv.URL+"/go", 30, srv.Client(), "test", false)
	if err != nil {
		t.Fatal(err)
	}
	if want := srv.URL + "/posts/article/"; finalURL != want {
		t.Errorf("final URL = %q, want %q", finalURL, want)
	}
}
