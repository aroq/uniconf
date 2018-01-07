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

func TestLoad(t *testing.T) {
	buffer := bytes.NewBuffer(testJsonConfig1)
	s := string(buffer.String())
	os.Setenv("UNIPIPE_CONFIG1", s)

	buffer = bytes.NewBuffer(testJsonConfig2)
	s = string(buffer.String())
	os.Setenv("UNIPIPE_CONFIG2", s)

	config := []map[string]interface{}{
		{
			"sourceName": "env",
			"sourceType": "env",
			"configs": []map[string]interface{}{
				{
					"id": "UNIPIPE_CONFIG1",
				},
				{
					"id": "UNIPIPE_CONFIG2",
				},
			},
		},
	}

	u = New()
	u.load(config)

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

