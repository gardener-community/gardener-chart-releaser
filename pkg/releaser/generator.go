package releaser

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
)


func fetchControllerRegistration(cfg SrcConfiguration) ([]byte, error) {
	var controller_registration []byte
	urls := [3]string{
		"https://raw.githubusercontent.com/" + cfg.Repo + "/" + cfg.Version + "/examples/controller-registration.yaml",
		"https://raw.githubusercontent.com/" + cfg.Repo + "/" + cfg.Version + "/example/controller-registration.yaml",
		"https://github.com/" + cfg.Repo + "/releases/download/" + cfg.Version + "/controller-registration.yaml",
	}

	for _, url := range urls {
		resp, err := http.Get(url)
		if err == nil {
			if resp.StatusCode == 200 {
				defer resp.Body.Close()
				controller_registration, err = ioutil.ReadAll(resp.Body)
				if err == nil {
					// Download was sucessful, return content
					logrus.Info("Successfully fetched chart for ", cfg.Name, " URL: ", url)
					return controller_registration, nil
				}
			}
		}
	}
	return nil, errors.New("Was not able to fetch chart for " + cfg.Name)
}


func generateExtensionChart(cfg SrcConfiguration) chart.Chart {
	controller_registration, err := fetchControllerRegistration(cfg)
	if err != nil {
		logrus.Warn(err.Error())
	}
	// add a helm template for values in the ControllerDeployment part of the chart
	controller_registration_as_string := string(controller_registration)
	controller_registration_split := strings.Split(controller_registration_as_string, "---")
	for i := range controller_registration_split {
		if strings.Contains(controller_registration_split[i], "ControllerDeployment") {
			controller_registration_split[i] += `{{- if .Values.values }}
{{- toYaml .Values.values | nindent 4 }}
{{- end }}
`
		}
	}

	controller_registration = []byte(strings.Join(controller_registration_split, "---"))

	// to create an empty values file:
	var values = make(map[string]interface{})
	values["values"] = map[string]interface{}{}
	values_serialized, _ := yaml.Marshal(values)

	controller_chart := chart.Chart{
		Metadata: &chart.Metadata{
			Name:        "controller",
			Version:     cfg.Version,
			Description: "Helmchart for controllerregistration of " + cfg.Name,
			APIVersion:  "v2",
		},
		Values: values,
		Raw: []*chart.File{{
			Name: "values.yaml",
			Data: values_serialized,
		}},
		Templates: []*chart.File{{
			Name: "templates/controller-registration.yaml",
			Data: controller_registration,
		}},
	}

	return controller_chart

}
