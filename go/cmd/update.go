package cmd

import (
	"fmt"
	"strings"

	"github.com/defer-ai/cli/internal/update"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update defer to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		currentVersion := strings.TrimPrefix(Version, "v")
		fmt.Printf("Current version: v%s\n", currentVersion)

		if currentVersion == "dev" {
			fmt.Println("Development build — skipping update check.")
			return nil
		}

		fmt.Println("Checking for updates...")

		release, err := update.FetchLatestRelease()
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}

		latest := strings.TrimPrefix(release.TagName, "v")

		if update.CompareVersions(latest, currentVersion) <= 0 {
			fmt.Printf("Already up to date (v%s)\n", currentVersion)
			return nil
		}

		fmt.Printf("New version available: v%s\n", latest)

		assetURL := update.FindAssetURL(release.Assets)
		if assetURL == "" {
			return fmt.Errorf("no release asset found for your platform (%s)\nVisit %s to download manually", update.AssetName(), release.HTMLURL)
		}

		fmt.Printf("Downloading %s...\n", update.AssetName())

		if err := update.DownloadAndReplace(assetURL); err != nil {
			return fmt.Errorf("update failed: %w", err)
		}

		fmt.Printf("Updated to v%s\n", latest)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
