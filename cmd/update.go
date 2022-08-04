package cmd

import (
	"fmt"

	"github.com/gardener-community/gardener-chart-releaser/pkg/releaser"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update all chart releases in the destination Github Repository",
	Long: `Running this requires the environment variable GITHUB_TOKEN to be set,
so that the program can interact with the GitHub API and create releaes in the
destination repository. 

Generally, the program will check, whether releases on the source side exist,
which are not availabe on the destination side, yet. If so, the missing releases
will be created. As of now, only releases will be tracked that are younger than the
maximum minor version minus 3.`,
	Run: func(cmd *cobra.Command, args []string) {

		config := releaser.Configuration{}
		viper.Unmarshal(&config) 

		ghToken := viper.GetString("GITHUB_TOKEN")
		targetDir := viper.GetString("targetDir")

		releaser.UpdateReleases(config, targetDir, ghToken)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().String("targetDir", "charts", "The directory where charts are stored locally")

	// add flags to viper according to
	// https://github.com/helm/chart-releaser/blob/main/pkg/config/config.go
	updateCmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flagName := flag.Name
		if flagName != "config" && flagName != "help" {
			if err := viper.BindPFlag(flagName, flag); err != nil {
				// can't really happen
				panic(fmt.Sprintln(errors.Wrapf(err, "Error binding flag '%s'", flagName)))
			}
		}
	})

}
