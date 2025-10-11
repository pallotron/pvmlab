package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// DownloadFile downloads a file from a URL to a local path.
func DownloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// DownloadImageIfNotExists checks if an image exists and downloads it if it doesn't.
func DownloadImageIfNotExists(imagePath, imageUrl string) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Checking for Ubuntu cloud image at %s...", imageUrl)
	s.Start()

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		s.Suffix = fmt.Sprintf(" Downloading Ubuntu cloud image from %s...", imageUrl)
		if err := DownloadFile(imagePath, imageUrl); err != nil {
			s.FinalMSG = color.RedString("✖ Failed to download Ubuntu cloud image.\n")
			s.Stop()
			return err
		}
		s.FinalMSG = color.GreenString("✔ Ubuntu cloud image downloaded successfully.\n")
	} else {
		s.FinalMSG = color.GreenString("✔ Ubuntu cloud image already exists.\n")
	}
	s.Stop()
	return nil
}
