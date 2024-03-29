package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"k8s.io/apimachinery/pkg/util/version"
)

func getLatestReleaseVersion() (string, error) {
	url := "https://api.github.com/repos/flomesh-io/fsm/releases/latest"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create GET request for latest release version from %s: %w", url, err)
	}

	req.Header.Add("Accept", "application/vnd.github.v3+json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to fetch latest release version from %s: %w", url, err)
	}
	//nolint: errcheck
	//#nosec G307
	defer resp.Body.Close()

	latestReleaseVersionInfo := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&latestReleaseVersionInfo); err != nil {
		return "", fmt.Errorf("unable to decode latest release version information from %s: %w", url, err)
	}

	latestVersion, ok := latestReleaseVersionInfo["tag_name"]
	if !ok {
		return "", fmt.Errorf("tag_name key not found in latest release version information from %s", url)
	}
	return fmt.Sprint(latestVersion), nil
}

func outputLatestReleaseVersion(out io.Writer, latestRelease string, currentRelease string) error {
	latest, err := version.ParseSemantic(latestRelease)
	if err != nil {
		return err
	}
	current, err := version.ParseSemantic(currentRelease)
	if err != nil {
		return err
	}
	if current.LessThan(latest) {
		fmt.Fprintf(out, "\nFSM %s is now available. Please see https://github.com/flomesh-io/fsm/releases/latest.\nWARNING: upgrading could introduce breaking changes. Please review the release notes.\n\n", latestRelease)
	}
	return nil
}
