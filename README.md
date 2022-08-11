# Gardener Chart releaser
In this repo, the code for a Gardener chart releaser is provided and maintained. This software aims at collecting and releasing charts required for Gardener provisioning, so that they can be accessed at one single point of truth. Releases are published on GitHub and a corresponding helm repository is configured via GitHub pages. Consequently, working with the released charts is as simple as it could be. Note that only the software itself is maintained here. The releases are/can be published in various other repositories (you could also setup your own, if you like).

# Getting Started

## The configuration file
First, please checkout the config.yaml file as a starting point. The overall structure is as follows:
``` yaml
destination:
  owner: gardener-community
  repo: gardener-charts

sources:
  - name: "gardener-controlplane"
    repo: "gardener/gardener"
    version: "v1.51.0"
    charts:
      - "charts/gardener/controlplane"
  - ...
```
`sources` defines a list with "upstream" charts to collect, and `destination` defines a repository (hosted on GitHub) serving as a helm repository where the charts are released.

## Export charts locally
If you want to export the configured charts to a local directory for development purposes, gardener-chart-releaser can do it for you. Simply run
```shell
go run main.go export
```
and find a `charts` directory containing the configured charts. Now, you can develop (with) these charts.

## Update the versions defined in config.yaml
You can simply update the versions in config.yaml to the latest version available upstream by
```shell
go run main.go fetchLatestVersions
```
This is useful in combination with exporting the charts to a local directory. If you fetch the lastest versions before, the charts in the local directory will also match the latest version.

## Further help
You can get further help by running the help commands implemented by the program. For instance,
```shell
go run main.go --help
```
or 
```shell
go run main.go update --help
```
will provide further information. 

# Contribute
Of course, you can contribute to this project. If something goes wrong, just file an issue. If you see room for improvements file a pull request. We will have a look at it. 
