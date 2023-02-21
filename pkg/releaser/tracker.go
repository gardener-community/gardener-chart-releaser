package releaser

import (
	"context"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/akrennmair/slice"
	"github.com/google/go-github/v36/github"
	"github.com/sirupsen/logrus"
)

func getReleasesToTrack(cfg SrcConfiguration, dst DstConfiguration, client *github.Client) ([]*semver.Version, error) {

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


	// As we release all charts in the 23ke-charts repo, we need to list way more releases.
	// Let's take the last 300 for now
	ourReleases := make([]*github.RepositoryRelease, 300)
	for i := 1; i <= 3; i++ {
		pageReleases, _, _ := client.Repositories.ListReleases(context.Background(),
			dst.Owner,
			dst.Repo,
			&github.ListOptions{
				Page:    i,
				PerPage: 100,
			})
		ourReleases = append(ourReleases, pageReleases...)
	}

	ourReleases = slice.Filter(ourReleases, (func(r *github.RepositoryRelease) bool {
		return strings.Contains(r.GetName(), cfg.Name)
	}))

	ourReleaseVersions := slice.Map(ourReleases, func(r *github.RepositoryRelease) *semver.Version {
		vAsStringSlice := strings.Split(r.GetTagName(), "-")
		v, _ := semver.NewVersion(vAsStringSlice[len(vAsStringSlice)-1])
		return v
	})

	// Now, filter out all version we have on our side.
	// If upstreamReleaseVersions is not empty afterwards,
	// we need to generate releases for these versions
	for _, ver := range ourReleaseVersions {
		upstreamReleaseVersions = slice.Filter(upstreamReleaseVersions, (func(v *semver.Version) bool {
			return !v.Equal(ver)
		}))
	}

	return upstreamReleaseVersions, nil

}
