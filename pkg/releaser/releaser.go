package releaser

import (
	"context"
	"os/exec"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/google/go-github/v36/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"helm.sh/helm/v3/pkg/chartutil"

	chartreleaserconfig "github.com/helm/chart-releaser/pkg/config"
	chartreleasergit "github.com/helm/chart-releaser/pkg/git"
	chartreleasergithub "github.com/helm/chart-releaser/pkg/github"
	chartreleaser "github.com/helm/chart-releaser/pkg/releaser"
)

// UpdateReleases ...
func UpdateReleases(config Configuration, targetDir string, ghToken string)  {

	// For updating the index.yaml on gh-pages, we need to run this in a corresponding gitrepo
	// Therefore, clone the destination repo and copy the .git folder to the current working dir

	// make sure to start in a clean state
	exec.Command("rm", "-rf", "destrepo").Run()
	exec.Command("rm", "-rf", ".git").Run()
	logrus.Info("Cloning destrepo ", config.DstCfg.Owner, "/", config.DstCfg.Repo)
	repo, err := git.PlainClone("destrepo", false, &git.CloneOptions{
		URL: "https://github.com/" + config.DstCfg.Owner + "/" + config.DstCfg.Repo,
		SingleBranch: false,
	})
	if err != nil {
		logrus.Error("Error during cloning of destination Repository: ", err)
		return
	}

	// check whether the gh-pages branch is availabe on the remote side
	// we need for the index.yaml
	remote, _ := repo.Remote("origin")
	rfs, _ := remote.List(&git.ListOptions{})
	ghPagesExists := false
	for _, r := range(rfs) {
		if strings.Contains(string(r.Name()), "gh-pages") {
			ghPagesExists = true
			break
		}
	}
	if ghPagesExists != true {
		logrus.Error("I cannot go on, as the gh-pages branch is not existing in your destination repo: ", err)
		return
	}

	// make sure the working dir is a git repository
	exec.Command("cp", "-R", "destrepo/.git", "./").Run()


	// get a *github.Client	for the github token
	// this client will be used for interacting with the github api
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)
	tokenClient := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tokenClient)

	// main loop over all items in the config file
	for _, cfg := range config.SrcCfg {
		versionsToRelease, _ := getReleasesToTrack(cfg, config.DstCfg, client)
		for _, v := range versionsToRelease {
			cfg.Version = v.Original()
			topLevelChart := getTopLevelChart(cfg, client)
			chartutil.Save(&topLevelChart, targetDir)
		}
	}

	// prepare the chart-releaser configuration
	chartrelcfg := chartreleaserconfig.Options{
		Owner:               config.DstCfg.Owner,
		GitRepo:             config.DstCfg.Repo,
		ChartsRepo:          config.DstCfg.Repo,
		IndexPath:           "destrepo/index.yaml",
		PagesIndexPath:      "index.yaml",
		PagesBranch:         "gh-pages",
		Remote:              "origin",
		PackagePath:         targetDir,
		Sign:                false,
		Token:               ghToken,
		Commit:              "",
		Push:                true,
		PR:                  false,
		SkipExisting:        true,
		ReleaseNameTemplate: "{{ .Name }}-{{ .Version }}",
		ReleaseNotesFile:    "RELEASE.md",
	}

	// define the chart releaser
	gh := chartreleasergithub.NewClient(chartrelcfg.Owner, chartrelcfg.GitRepo, ghToken, "https://api.github.com/", "https://uploads.github.com/")
	releaser := chartreleaser.NewReleaser(&chartrelcfg, gh, &chartreleasergit.Git{})

	err = releaser.CreateReleases()
	_, err = releaser.UpdateIndexFile()

	// cleanup
	exec.Command("rm", "-rf", "destrepo").Run()
	exec.Command("rm", "-rf", ".git").Run()

}
