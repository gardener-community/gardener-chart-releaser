package releaser

import (
	"context"
	"gopkg.in/yaml.v3"
	"os"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/akrennmair/slice"
	"github.com/google/go-github/v36/github"
	"github.com/sirupsen/logrus"
)

func getReleasesToTrack(cfg SrcConfiguration, dst DstConfiguration, client *github.Client, indexYamlPath string) ([]*semver.Version, error) {

	owner := strings.Split(cfg.Repo, "/")[0]
	repo := strings.Split(cfg.Repo, "/")[1]

	// most probably the last 20 upstreamReleases will contain everything we need
	// assuming that we do not have more than 5 patch releaeses in 4 consecutive
	// minor tracks
	upstreamReleases, _, err := client.Repositories.ListReleases(context.Background(),
		owner,
		repo,
		&github.ListOptions{
			Page:    0,
			PerPage: 20,
		})

	if err != nil {
		logrus.Warn(err.Error())
	}
	// get and sort upstream release versions
	upstreamReleaseVersions := make([]*semver.Version, len(upstreamReleases))
	for i, r := range upstreamReleases {
		v, err := semver.NewVersion(r.GetTagName())
		if err != nil {
			return nil, err
		}
		upstreamReleaseVersions[i] = v
	}
	sort.Sort(semver.Collection(upstreamReleaseVersions))

	index := make(map[string]any)
	err = readYamlFile(indexYamlPath, index)
	if err != nil {
		return nil, err
	}

	entries := index["entries"].(map[string]any)[cfg.Name].([]any)
	ourReleaseVersions := slice.Map(entries, func(e any) *semver.Version {
		eMap := e.(map[string]any)

		versionString := eMap["version"].(string)
		version, err := semver.NewVersion(versionString)
		if err != nil {
			logrus.Warn(err)
		}
		return version
	})

	// Now, filter out all version we have on our side.
	// If upstreamReleaseVersions is not empty afterwards,
	// we need to generate releases for these versions
	for _, ver := range ourReleaseVersions {
		upstreamReleaseVersions = slice.Filter(upstreamReleaseVersions, func(v *semver.Version) bool {
			return !v.Equal(ver)
		})
	}

	return upstreamReleaseVersions, nil

}

func readYamlFile(path string, out any) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(bytes, out)
	if err != nil {
		return err
	}

	return nil
}
