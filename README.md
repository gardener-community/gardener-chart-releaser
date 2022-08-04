# Gardener Chart releaser
In this repo, the code for a Gardener chart releaser is provided and maintained.

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
