package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file from %s: %s", url, resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// DownloadImageIfNotExists checks if an image exists and downloads it if it doesn't.
var DownloadImageIfNotExists = func(imagePath, imageUrl string) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Checking for Ubuntu cloud image at %s...", imageUrl)
	s.Start()

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		s.Suffix = fmt.Sprintf(color.CyanString(" Downloading Ubuntu cloud image from %s..."), imageUrl)
		if err := DownloadFile(imagePath, imageUrl); err != nil {
			s.Stop()
			fmt.Printf("%s %s\n", color.RedString("✖"), s.Suffix)
			return err
		}
		s.Stop()
		fmt.Printf("%s %s\n", color.GreenString("✔"), strings.TrimLeft(s.Suffix, " "))
	} else {
		s.Stop()
		fmt.Printf("%s %s\n", color.GreenString("✔"), s.Suffix)
	}
	return nil
}
