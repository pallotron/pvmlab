package downloader

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDownloadFile_Success(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	// Create a temporary file to download to
	tmpfile, err := ioutil.TempFile("", "download-test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	// Call the function under test
	err = DownloadFile(tmpfile.Name(), server.URL)
	if err != nil {
		t.Fatalf("DownloadFile() returned an error: %v", err)
	}

	// Verify the content of the downloaded file
	content, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("Downloaded file content = %q, want %q", string(content), "test content")
	}
}

func TestDownloadFile_ServerError(t *testing.T) {
	// Create a mock server that always returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create a temporary file to download to
	tmpfile, err := ioutil.TempFile("", "download-test-fail")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	// Call the function under test
	err = DownloadFile(tmpfile.Name(), server.URL)
	if err == nil {
		t.Fatal("DownloadFile() did not return an error for a server error")
	}
}
