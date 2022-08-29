package releaser

import (
	"context"
	"fmt"
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

func importChart(cfg SrcConfiguration, src string) chart.Chart {

	logrus.Info("Starting Chart import from ", cfg.Repo, " Version: ", cfg.Version)
	tempRepoDir := "/tmp/" + cfg.Repo + "/"
	_, err := git.PlainClone(tempRepoDir, false, &git.CloneOptions{
		URL:           "https://github.com/" + cfg.Repo,
		ReferenceName: plumbing.NewTagReferenceName(cfg.Version),
		SingleBranch:  true,
		Depth:         1,
		Progress:      os.Stdout,
	})
	if err != nil {
		fmt.Println(err)
	}

	// I did not find any package handling the symlinks to directories,
	// so that the directories are copied over
	// Therefore, just use a system command here
	tempDir := "tmp"

	_, err = exec.Command("cp", "-LR", tempRepoDir+src, tempDir).Output()
	if err != nil {
		fmt.Println(err)
	}

	resultChart, err := loader.Load(tempDir)
	if err != nil {
		fmt.Println(err)
	}
	err = os.RemoveAll(tempDir)
	if err != nil {
		logrus.Error(err)
	}
	resultChart.Metadata.Version = cfg.Version

	return *resultChart
}

func ensureChart(c *chart.Chart, cfg SrcConfiguration) {

	c.Metadata.APIVersion = "v2"

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

		// helmcharts are versioned with strict semver (no v-Prefix)
		re := regexp.MustCompile(`^v`)
		c.Metadata.Version = string(re.ReplaceAll([]byte(cfg.Version), []byte("")))

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

func getTopLevelChart(cfg SrcConfiguration, client *github.Client) chart.Chart {

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
				*subChart = importChart(cfg, src)
			}
			mainChart.AddDependency(subChart)
		}

	} else {
		// here we assume that the chart is already packaged appropriately by upstream
		mainChart = importChart(cfg, cfg.Charts[0])
	}

	// ensureChart makes sure that the chart dependencies are set correctly
	mainChart.Metadata.Name = cfg.Name
	mainChart.Files = append(mainChart.Files, writeReleaseNotes(cfg, client))
	ensureChart(&mainChart, cfg)
	return mainChart
}
