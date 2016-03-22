package testserver

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	awsBaseURL      = "https://s3.amazonaws.com/cockroach/cockroach"
	latestSuffix    = "LATEST"
	localBinaryPath = "/var/tmp"
)

func binaryName() string {
	return fmt.Sprintf("cockroach.%s-%s", runtime.GOOS, runtime.GOARCH)
}

func binaryNameWithSha(sha string) string {
	return fmt.Sprintf("%s.%s", binaryName(), sha)
}

func binaryPath(sha string) string {
	return filepath.Join(localBinaryPath, binaryNameWithSha(sha))
}

func latestMarkerURL() string {
	return fmt.Sprintf("%s/%s.%s", awsBaseURL, binaryName(), latestSuffix)
}

func binaryURL(sha string) string {
	return fmt.Sprintf("%s/%s.%s", awsBaseURL, binaryName(), sha)
}

func findLatestSha() (string, error) {
	markerURL := latestMarkerURL()
	marker, err := http.Get(markerURL)
	if err != nil {
		return "", fmt.Errorf("could not download %s: %s", markerURL)
	}
	if marker.StatusCode == 404 {
		return "", fmt.Errorf("for 404 from GET %s: make sure OS and ARCH are supported",
			markerURL)
	} else if marker.StatusCode != 200 {
		return "", fmt.Errorf("bad response got GET %s: %d (%s)",
			markerURL, marker.StatusCode, marker.Status)
	}

	defer marker.Body.Close()
	body, err := ioutil.ReadAll(marker.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func downloadFile(url, filePath string) error {
	output, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("error creating %s: %s", filePath, "-", err)
	}
	defer output.Close()

	log.Printf("downloading %s to %s, this may take some time", url, filePath)

	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading %s: %s", url, err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("error downloading %s: %d (%s)", url, response.StatusCode, response.Status)
	}

	_, err = io.Copy(output, response.Body)
	if err != nil {
		return fmt.Errorf("problem downloading %s to %s: %s", url, filePath, err)
	}
	return nil
}

func downloadLatestBinary() (string, error) {
	sha, err := findLatestSha()
	if err != nil {
		return "", err
	}

	localFile := binaryPath(sha)
	if _, err := os.Stat(localFile); err == nil {
		// File already present.
		return localFile, nil
	}

	err = downloadFile(binaryURL(sha), localFile)
	if err != nil {
		_ = os.Remove(localFile)
		return "", err
	}

	return localFile, nil
}
