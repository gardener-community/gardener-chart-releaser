package releaser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-github/v36/github"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
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

	_, err = exec.Command("cp", "-Lr", tempRepoDir+src, tempDir).Output()
	if err != nil {
		fmt.Println(err)
	}

	resultChart, err := loader.Load(tempDir)
	if err != nil {
		fmt.Println(err)
	}
	os.RemoveAll(tempDir)
	resultChart.Metadata.Version = cfg.Version

	return *resultChart
}

func ensureChart(c *chart.Chart) {

	c.Metadata.APIVersion = "v2"
	if len(c.Dependencies()) == 0 {
		return
	}

	for _, dep := range c.Dependencies() {
		cur_dep := chart.Dependency{
			Name:      dep.Name(),
			Condition: dep.Name() + ".enabled",
			Enabled:   false,
		}
		c.Metadata.Dependencies = append(c.Metadata.Dependencies, &cur_dep)

		if c.Values == nil {
			c.Values = make(map[string]interface{})
		}
		c.Values[dep.Name()] = map[string]bool{"enabled": false}

		values_serialized, _ := yaml.Marshal(c.Values)
		c.Raw = []*chart.File{{
			Name: "values.yaml",
			Data: values_serialized,
		}}

		ensureChart(dep)
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
	ensureChart(&mainChart)
	return mainChart
}
