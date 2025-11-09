package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cheggaaa/pb/v3"
	"github.com/fatih/color"
)

// DownloadFile downloads a file from a URL to a local path, with support for Range headers and a progress bar.
func DownloadFile(path string, url string, rangeHeader string, totalSize int64, initialSize int64) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if rangeHeader != "" {
		req.Header.Add("Range", rangeHeader)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("failed to download file from %s: %s", url, resp.Status)
	}

	// Ensure the destination directory exists.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	openFlags := os.O_CREATE | os.O_WRONLY
	if rangeHeader != "" {
		openFlags |= os.O_APPEND
	} else {
		openFlags |= os.O_TRUNC
	}

	out, err := os.OpenFile(path, openFlags, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	bar := pb.New64(totalSize)
	bar.SetCurrent(initialSize)
	bar.Set(pb.Bytes, true)
	bar.SetTemplateString(`{{counters . }} {{bar . }} {{percent . }} {{rtime . }} {{speed . }}`)
	bar.SetWidth(80)
	bar.Start()

	barReader := bar.NewProxyReader(resp.Body)
	defer barReader.Close()

	if _, err = io.Copy(out, barReader); err != nil {
		bar.Finish()
		return err
	}

	bar.Finish()
	return nil
}

// DownloadImageIfNotExists checks if an image exists and downloads it if it doesn't,
// supporting resumable downloads.
var DownloadImageIfNotExists = func(imagePath, imageUrl string) error {
	color.Cyan("i Checking for distro image at %s...", imageUrl)

	// Get remote file size
	resp, err := http.Head(imageUrl)
	if err != nil {
		return fmt.Errorf("failed to get remote file headers: %w", err)
	}
	remoteSize := resp.ContentLength

	localFileInfo, err := os.Stat(imagePath)
	if err == nil {
		// File exists, check size
		localSize := localFileInfo.Size()
		if remoteSize > 0 && localSize == remoteSize {
			color.Green("✔ Distro image is already up to date.")
			return nil
		}
		if remoteSize > 0 && localSize < remoteSize {
			// Resume download
			color.Cyan("i Resuming download of distro image from %s...", imageUrl)
			rangeHeader := fmt.Sprintf("bytes=%d-", localSize)
			if err := DownloadFile(imagePath, imageUrl, rangeHeader, remoteSize, localSize); err != nil {
				return err
			}
			color.Green("✔ Download complete.")
			return nil
		}
		// Local file is larger or remote size is unknown, re-download
	}

	// File does not exist or needs re-downloading
	color.Cyan("i Downloading distro image from %s...", imageUrl)
	if err := DownloadFile(imagePath, imageUrl, "", remoteSize, 0); err != nil {
		return err
	}
	color.Green("✔ Download complete.")

	return nil
}

