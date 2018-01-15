package uniconf

import (
	"testing"
	"os"
	"bytes"
)

var testJsonConfig1 = []byte(`{
  "log_level": "TRACE"
}`)

var testJsonConfig2 = []byte(`{
  "log_level": "INFO"
}`)

var testJsonConfig3 = []byte(`{
  "sources": {
    "drupipe": {
      "repo": "https://github.com/aroq/drupipe-scenarios.git",
      "ref": "actions-dev",
      "prefix": "refs/heads/"
    }
  },
  "log_level": "INFO"
}`)

func LoadTestConfig() {
	u = New()
	AddConfigProvider(testUniconfConfig)
	AddPhase(&Phase{
		Name: "load_config",
		Callbacks: []*Callback{
			{
				Args:   nil,
				Method: Load,
			},
		},
	})
	AddPhase(&Phase{
		Name: "process_contexts",
		Callbacks: []*Callback{
			{
				Args:   nil,
				Method: ProcessContexts,
			},
		},
	})
	AddPhase(&Phase{
		Name: "flatten_config",
		Callbacks: []*Callback{
			{
				Args:   nil,
				Method: FlattenConfig,
			},
		},
	})

	Execute()
}

// defaultUniconfConfig provides default Uniconf configuration.
func testUniconfConfig() interface{} {
	buffer := bytes.NewBuffer(testJsonConfig1)
	s := string(buffer.String())
	os.Setenv("UNIPIPE_CONFIG1", s)

	buffer = bytes.NewBuffer(testJsonConfig2)
	s = string(buffer.String())
	os.Setenv("UNIPIPE_CONFIG2", s)

	return map[string]interface{}{
		"sources": map[string]interface{}{
			"env": map[string]interface{}{
				"type": "env",
			},
		},
		"from": []interface{}{
			"env:UNIPIPE_CONFIG1",
			"env:UNIPIPE_CONFIG2",
		},
	}
}

func TestLoad(t *testing.T) {
	LoadTestConfig()

	//t.Logf("Config: \n%v", u.config)
	//t.Logf("History: \n%v", u.history)

	if u.config["log_level"].(string) != "INFO" {
		t.Errorf("Uniconf load failed: log_level=INFO")
	}

	if _, ok := u.history["log_level"].(map[string]interface{})["load"].(map[string]interface{})["env:UNIPIPE_CONFIG1"]; !ok {
		t.Errorf("Uniconf history failed env:UNIPIPE_CONFIG1")
	}
	if _, ok := u.history["log_level"].(map[string]interface{})["load"].(map[string]interface{})["env:UNIPIPE_CONFIG2"]; !ok {
		t.Errorf("Uniconf history failed env:UNIPIPE_CONFIG2")
	}
}

func TestInterpolateString(t *testing.T) {
	LoadTestConfig()
	//src, _ := unitool.UnmarshalYaml(yamlExample)

	result := InterpolateString("${log_level}", u.flatConfig)
	//result := InterpolateString("${params.jobs.dev.params.branch}", u.flatConfig)
	if result != "INFO" {
		t.Errorf("Interpolate string failed: expected value: 'master', real value: %v", result)
	}

	result = InterpolateString("${deepGet(\"log_level\")}", u.flatConfig)
	//result = InterpolateString("${deepGet(\"params.jobs.dev.params.branch\")}", u.flatConfig)
	if result != "INFO" {
		t.Errorf("Interpolate string with initial deepGet() failed: expected value: 'master', real value: %v", result)
	}
}

var yamlExample = []byte(`params:
  jobs:
    params:
      jobs_param: true
    dev:
      params:
        jobs_param: false
        jobs_dev_param: true
        branch: master
        context:
          environment: dev
    prod:
      params:
        branch: master
        context:
          environment: prod
`)


