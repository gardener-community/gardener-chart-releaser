package releaser

import (
	"context"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-github/v36/github"
	"github.com/sirupsen/logrus"
	"github.com/tomwright/dasel"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func importChart(cfg SrcConfiguration, src string) (chart.Chart, error) {

	// I did not find any package handling the symlinks to directories,
	// so that the directories are copied over
	// Therefore, just use a system command here
	tempDir := "tmp"
	tempRepoDir := "/tmp/" + cfg.Repo + "/"
	logrus.Info("Git clone or pull: ", cfg.Repo, " Version: ", cfg.Version, " tmp-dir: ", tempRepoDir)

	_, err := exec.Command("rm", "-rf", tempDir).Output()
	if err != nil {
		logrus.Warn(err)
	}

	// Clone the repository or open it, if it already exists on disk
	// It is handeled like this for performance reasons, when e.g. exporting the charts
	repo, err := git.PlainClone(tempRepoDir, false, &git.CloneOptions{
		URL: "https://github.com/" + cfg.Repo,
	})
	if err == git.ErrRepositoryAlreadyExists {
		repo, err = git.PlainOpen(tempRepoDir)
	} else if err != nil {
		logrus.Info(err)
	}

	// We want to checkout the "default" branch of our repo, so that a clean state is reached and we can pull
	// Therefore, find out if the "default" branch is called "master" oder "main"
	b, err := repo.Branches()
	var branch *plumbing.Reference
	for {
		branch, err = b.Next()
		if err != nil {
			logrus.Error("I was not able to find a default branch, you should not rely on what I will do next")
			break
		}
		if strings.Contains(string(branch.Name()), "master") || strings.Contains(string(branch.Name()), "main") {
			break
		}
	}

	// checkout the default branch now and pull
	wt, err := repo.Worktree()
	wt.Checkout(&git.CheckoutOptions{
		Branch: branch.Name(),
	})
	err = wt.Pull(&git.PullOptions{})
	if err != nil {
		logrus.Info(err)
	}

	// checkout the target-tag after pull
	err = wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewTagReferenceName(cfg.Version),
	})
	if err != nil {
		logrus.Warn(err)
	}

	_, err = exec.Command("cp", "-LR", tempRepoDir+src, tempDir).Output()
	if err != nil {
		logrus.Warn(err)
	}

	resultChart, err := loader.Load(tempDir)
	if err != nil {
		logrus.Warn(err)
		return chart.Chart{}, err
	}
	err = os.RemoveAll(tempDir)
	if err != nil {
		logrus.Warn(err)
	}
	resultChart.Metadata.Version = cfg.Version

	return *resultChart, nil
}

func ensureChart(c *chart.Chart, cfg SrcConfiguration) {

	c.Metadata.APIVersion = "v2"

	// helmcharts are versioned with strict semver (no v-Prefix)
	re := regexp.MustCompile(`^v`)
	c.Metadata.Version = string(re.ReplaceAll([]byte(cfg.Version), []byte("")))

	rootNode := dasel.New(c.Values)
	switch c.Name() {
	case "dashboard":
		err := rootNode.Put("image.tag", cfg.Version)
		if err != nil {
			logrus.Error(err)
		}
	case "gardenlet":
		err := rootNode.Put("global.gardenlet.image.tag", cfg.Version)
		if err != nil {
			logrus.Error(err)
		}
	case "gardener-controlplane":
		err := rootNode.Put("global.apiserver.image.tag", cfg.Version)
		if err != nil {
			logrus.Error(err)
		}
		err = rootNode.Put("global.admission.image.tag", cfg.Version)
		if err != nil {
			logrus.Error(err)
		}
		err = rootNode.Put("global.controller.image.tag", cfg.Version)
		if err != nil {
			logrus.Error(err)
		}
		err = rootNode.Put("global.scheduler.image.tag", cfg.Version)
		if err != nil {
			logrus.Error(err)
		}
	}
	valuesSerialized, err := yaml.Marshal(rootNode.OriginalValue)
	if err != nil {
		logrus.Error(err)
	}
	c.Raw = []*chart.File{{
		Name: "values.yaml",
		Data: valuesSerialized,
	}}

	if len(c.Dependencies()) == 0 {
		return
	}

	for _, dep := range c.Dependencies() {
		curDep := chart.Dependency{
			Name:      dep.Name(),
			Condition: dep.Name() + ".enabled",
			Enabled:   false,
		}
		c.Metadata.Dependencies = append(c.Metadata.Dependencies, &curDep)

		if c.Values == nil {
			c.Values = make(map[string]interface{})
		}
		c.Values[dep.Name()] = map[string]bool{"enabled": false}

		valuesSerialized, err := yaml.Marshal(c.Values)
		if err != nil {
			logrus.Error(err)
		}
		c.Raw = []*chart.File{{
			Name: "values.yaml",
			Data: valuesSerialized,
		}}

		ensureChart(dep, cfg)
	}
}

func writeReleaseNotes(cfg SrcConfiguration, client *github.Client) *chart.File {
	rr, _, _ := client.Repositories.GetReleaseByTag(context.Background(), strings.Split(cfg.Repo, "/")[0], strings.Split(cfg.Repo, "/")[1], cfg.Version)

	file := &chart.File{
		Name: "RELEASE.md",
		Data: []byte(*rr.Body),
	}
	return file
}

func getTopLevelChart(cfg SrcConfiguration, client *github.Client) (chart.Chart, error) {

	var mainChart chart.Chart

	// We need to generate a new Chart, when controller-registrations are involved, as extension
	// controllers are not packaged as charts by upstream
	generateNewChart := false
	for _, src := range cfg.Charts {
		if src == "controller-registration" {
			generateNewChart = true
		}
		break
	}

	var err error
	if generateNewChart {
		mainChart = chart.Chart{
			Metadata: &chart.Metadata{
				Name:        cfg.Name,
				Version:     cfg.Version,
				Description: "A helmchart for " + cfg.Name,
				APIVersion:  "v2",
			},
		}

		// if src equals "controller-registration", we need to generate the Chart for
		// the controller of an extension
		for _, src := range cfg.Charts {
			subChart := new(chart.Chart)
			if src == "controller-registration" {
				*subChart = generateExtensionChart(cfg)
			} else {
				*subChart, err = importChart(cfg, src)
				if err != nil {
					continue
				}
			}
			mainChart.AddDependency(subChart)
		}

	} else {
		// here we assume that the chart is already packaged appropriately by upstream
		mainChart, err = importChart(cfg, cfg.Charts[0])
		if err != nil {
			return chart.Chart{}, err
		}
	}

	// ensureChart makes sure that the chart dependencies are set correctly
	mainChart.Metadata.Name = cfg.Name
	mainChart.Files = append(mainChart.Files, writeReleaseNotes(cfg, client))
	ensureChart(&mainChart, cfg)
	return mainChart, nil
}
