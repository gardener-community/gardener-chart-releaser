/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/gardener-community/gardener-chart-releaser/pkg/releaser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Exports charts to a directory (requires GITHUB_TOKEN)",
	Long:  `This is a utility command which can be used for generating
a local development version of the charts. Once exported the charts
can be modified and tested. If you feel confident that your changes
also make sense for others you can go ahead and file pull requests in
the corresponding upstream repository.

This command requires the environmet variable GITHUB_TOKEN to be set.`,
	Run: func(cmd *cobra.Command, args []string) {

		config := releaser.Configuration{}
		viper.Unmarshal(&config)
		ghToken := viper.GetString("GITHUB_TOKEN")
		targetDir := viper.GetString("targetDir")

		releaser.ExportCharts(config, targetDir, ghToken)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().String("targetDir", "charts", "The directory where charts are stored locally")
}
