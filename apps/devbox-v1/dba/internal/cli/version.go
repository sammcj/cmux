// internal/cli/version.go
package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"

	// Flag for version check
	versionCheckFlag bool
)

// LatestReleaseURL is the URL to check for the latest release
// Can be overridden for testing
var LatestReleaseURL = "https://api.github.com/repos/dba-cli/dba/releases/latest"

func SetVersionInfo(v, c, bt string) {
	version = v
	commit = c
	buildTime = bt
}

// GetVersion returns the current version
func GetVersion() string {
	return version
}

type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func (v VersionInfo) TextOutput() string {
	return fmt.Sprintf("dba version %s (commit: %s, built: %s)\nGo: %s %s/%s",
		v.Version, v.Commit, v.BuildTime, v.GoVersion, v.OS, v.Arch)
}

// VersionCheckResult contains the result of checking for updates
type VersionCheckResult struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version,omitempty"`
	UpdateAvail    bool   `json:"update_available"`
	DownloadURL    string `json:"download_url,omitempty"`
	Error          string `json:"error,omitempty"`
}

func (v VersionCheckResult) TextOutput() string {
	if v.Error != "" {
		return fmt.Sprintf("Current version: %s\nError checking for updates: %s", v.CurrentVersion, v.Error)
	}
	if v.UpdateAvail {
		return fmt.Sprintf("Current version: %s\nLatest version:  %s\n\nUpdate available! Download from: %s",
			v.CurrentVersion, v.LatestVersion, v.DownloadURL)
	}
	return fmt.Sprintf("Current version: %s\nYou are running the latest version.", v.CurrentVersion)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long: `Print version information for the DBA CLI.

Use --check to check if a newer version is available.`,
	Run: func(cmd *cobra.Command, args []string) {
		if versionCheckFlag {
			checkForUpdates()
			return
		}

		info := VersionInfo{
			Version:   version,
			Commit:    commit,
			BuildTime: buildTime,
			GoVersion: runtime.Version(),
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
		}
		OutputResult(info)
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionCheckFlag, "check", false,
		"Check if a newer version is available")
}

// checkForUpdates checks GitHub for the latest release
func checkForUpdates() {
	result := VersionCheckResult{
		CurrentVersion: version,
	}

	// Skip check for dev version
	if version == "dev" {
		result.Error = "cannot check updates for development version"
		OutputResult(result)
		return
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(LatestReleaseURL)
	if err != nil {
		result.Error = fmt.Sprintf("failed to check for updates: %v", err)
		OutputResult(result)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("failed to check for updates: HTTP %d", resp.StatusCode)
		OutputResult(result)
		return
	}

	var release struct {
		TagName    string `json:"tag_name"`
		HTMLURL    string `json:"html_url"`
		Name       string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		result.Error = fmt.Sprintf("failed to parse release info: %v", err)
		OutputResult(result)
		return
	}

	// Compare versions (simple string comparison, assumes semver)
	latestVersion := release.TagName
	if len(latestVersion) > 0 && latestVersion[0] == 'v' {
		latestVersion = latestVersion[1:] // Remove 'v' prefix
	}

	result.LatestVersion = latestVersion
	result.DownloadURL = release.HTMLURL
	result.UpdateAvail = latestVersion != version && latestVersion > version

	OutputResult(result)
}
