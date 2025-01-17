package commands

import (
	"github.com/Nightapes/go-semantic-release/pkg/semanticrelease"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	changelogCmd.Flags().Bool("checks", false, "Check for missing values and envs")
	changelogCmd.Flags().StringP("out", "o", "CHANGELOG.md", "Name of the file")
	rootCmd.AddCommand(changelogCmd)
}

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Generate changelog and save to file",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := cmd.Flags().GetString("config")
		if err != nil {
			return err
		}

		repository, err := cmd.Flags().GetString("repository")
		if err != nil {
			return err
		}

		force, err := cmd.Flags().GetBool("no-cache")
		if err != nil {
			return err
		}

		file, err := cmd.Flags().GetString("out")
		if err != nil {
			return err
		}

		configChecks, err := cmd.Flags().GetBool("checks")
		if err != nil {
			return err
		}

		s, err := semanticrelease.New(readConfig(config), repository, configChecks)
		if err != nil {
			return err
		}

		provider, err := s.GetCIProvider()
		if err != nil {
			return err
		}

		releaseVersion, err := s.GetNextVersion(provider, force)
		if err != nil {
			return err
		}
		log.Debugf("Found %d commits till last release", len(releaseVersion.Commits))

		generatedChangelog, err := s.GetChangelog(releaseVersion)
		if err != nil {
			return err
		}

		return s.WriteChangeLog(generatedChangelog.Content, file)
	},
}
