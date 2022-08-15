package cmd

import (
	"context"
	"log"
	"strings"

	"github.com/gardener-community/gardener-chart-releaser/pkg/releaser"
	"github.com/google/go-github/v36/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// fetchLatestVersionsCmd represents the fetchLatestVersions command
var fetchLatestVersionsCmd = &cobra.Command{
	Use:   "fetchLatestVersions",
	Short: "Fetches the latest versions from upstream and writes in the config file",
	Long: `This is a utility command for updating all versions specified in config.yaml.
It comes handy, when charts are exported for development purposes and one wants to
export the most recent version.
`,
	Run: func(cmd *cobra.Command, args []string) {

		config := releaser.Configuration{}
		viper.Unmarshal(&config)
		ghToken := viper.GetString("GITHUB_TOKEN")
		if ghToken == "" {
			log.Fatal("GITHUB_TOKEN is empty")
		}

		// get a *github.Client	for the github token
		// this client will be used for interacting with the github api
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: ghToken},
		)
		tokenClient := oauth2.NewClient(context.Background(), ts)
		client := github.NewClient(tokenClient)

		// main loop over all items in the config file
		for i, cfg := range config.SrcCfg {
			owner := strings.Split(cfg.Repo, "/")[0]
			repo := strings.Split(cfg.Repo, "/")[1]
			latestRelease, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
			if err != nil {
				log.Fatal(err)
			}
			config.SrcCfg[i].Version = *latestRelease.TagName
		}
		viper.Set("sources", config.SrcCfg)
		viper.WriteConfig()
	},
}

func init() {
	rootCmd.AddCommand(fetchLatestVersionsCmd)

}
