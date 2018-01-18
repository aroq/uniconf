package uniconf

import (
	"bytes"
	"github.com/aroq/uniconf/unitool"
	"os"
	"testing"
)

// TestMain provides config values and executes tests.
func PrepareTest() {
	u = New()

	// Set environment variables.
	var envVars = map[string][]byte{
		"UNICONF": []byte(`{
			"log_level": "INFO"
		}`),
		"UNICONF_TEST": []byte(`{
			"log_level": "TRACE"
		}`),
		// Actual value for from: env:UNICONF_TEST_MULTIPART_ENVVAR.
		"UNICONF_TEST_MULTIPART": []byte(`{
			"log_level": "DEBUG" 
		}`),
	}
	for k, v := range envVars {
		buffer := bytes.NewBuffer(v)
		s := string(buffer.String())
		os.Setenv(k, s)
	}

	// rootConfig provides default configuration.
	rootConfig := func() map[string]interface{} {
		return map[string]interface{}{
			"sources": map[string]interface{}{
				"env": map[string]interface{}{
					"type": "env",
				},
			},
			"from": []interface{}{
				"env:UNICONF",
				"project:root.yaml",
				"env:UNICONF_TEST_MULTIPART_ENVVAR",
			},
		}
	}

	// Provide sources & config entities.
	AddSource(NewSourceConfigMap("root", map[string]interface{}{
		"configMap": map[string]interface{}{
			"root": rootConfig(),
		},
	}))
	SetRootSource("root")

	AddSource(NewSourceConfigMap("project", map[string]interface{}{
		"configMap": map[string]interface{}{
			"root.yaml": testHelmProjectYaml,
		},
	}))

	AddSource(NewSourceConfigMap("drupipe", map[string]interface{}{
		"configMap": map[string]interface{}{
			"helm.yaml":        testDrupipeHelmYaml,
			"helm/blocks.yaml": testDrupipeHelmBlocksYaml,
			"helm/jobs.yaml":   testDrupipeHelmJobsYaml,
			"v3/actions.yaml":  testDrupipeV3ActionsYaml,
		},
	}))
}

// TestLoad tests config load & basic functions.
func TestLoad(t *testing.T) {
	PrepareTest()

	AddPhase(&Phase{
		Name: "config",
		Phases: []*Phase{
			{
				Name:     "load",
				Callback: Load,
			},
		},
	})

	SetContexts("jobs.dev.jobs.install")

	Execute()

	t.Run("u.config defined", func(t *testing.T) {
		if u.config == nil {
			t.Fatalf("Uniconf load failed: no config")
		}
	})

	t.Run("log_level", func(t *testing.T) {
		if _, ok := u.config["log_level"]; !ok {
			t.Fatalf("Uniconf load failed: no 'log_level' key in config")
		}
		t.Run("log_level==DEBUG", func(t *testing.T) {
			if u.config["log_level"].(string) != "DEBUG" {
				t.Errorf("Uniconf load failed: log_level=DEBUG")
			}
		})
		t.Run("history==env:UNICONF", func(t *testing.T) {
			if _, ok := u.history["log_level"].(map[string]interface{})["load"].(map[string]interface{})["env:UNICONF"]; !ok {
				t.Errorf("Uniconf history failed env:UNICONF")
			}
		})
	})

	t.Run("Collect(params.jobs.common.helm.install).pipeline.from = '.params.pipelines.helm.install'", func(t *testing.T) {
		path := "params.jobs.common.helm.install"
		params, _ := unitool.CollectKeyParamsFromJsonPath(u.config, path, "params")
		//t.Logf("collectKeyParamsFromJsonPath result: %v", params)

		path = "pipeline.from"
		value := ".params.pipelines.helm.install"
		result := unitool.SearchMapWithPathStringPrefixes(params, path)
		if result != value {
			t.Errorf("Deep key search failed: %s, expected value: %v, real value: %v", path, value, result)
		}
	})

}

// TestLoad tests config load & basic functions.
func TestLoadWithFlattenConfig(t *testing.T) {
	PrepareTest()

	AddPhase(&Phase{
		Name: "config",
		Phases: []*Phase{
			{
				Name:     "load",
				Callback: Load,
			},
			{
				Name:     "flatten_config",
				Callback: FlattenConfig,
			},
		},
	})

	SetContexts("jobs.dev.jobs.install")

	Execute()

	t.Run("InterpolateString", func(t *testing.T) {
		t.Run("${log_level}==DEBUG", func(t *testing.T) {
			result := InterpolateString("${log_level}", u.flatConfig)
			if result != "DEBUG" {
				t.Errorf("Interpolate string failed: expected value: 'master', real value: %v", result)
			}
		})
		t.Run("${deepGet(log_level)}==DEBUG", func(t *testing.T) {
			result := InterpolateString("${deepGet(\"log_level\")}", u.flatConfig)
			if result != "DEBUG" {
				t.Errorf("Interpolate string with initial deepGet() failed: expected value: 'master', real value: %v", result)
			}
		})
	})
}

// TestLoad tests config load & basic functions.
func TestLoadWithContexts(t *testing.T) {
	PrepareTest()

	AddPhase(&Phase{
		Name: "config",
		Phases: []*Phase{
			{
				Name:     "load",
				Callback: Load,
			},
			{
				Name:     "process_contexts",
				Callback: ProcessContexts,
			},
		},
	})

	SetContexts("jobs.dev.jobs.install")

	Execute()

	t.Run("u.config defined", func(t *testing.T) {
		if u.config == nil {
			t.Fatalf("Uniconf load failed: no config")
		}
	})
}

// Test config entities.
var testHelmProjectYaml = []byte(`---
from:
  - drupipe:helm.yaml
container_types:
  helm:
    apply:
      type: chart
jobs:
  dev: null
  preprod: null
  merge-requests: null
  prod:
    from:
      - .params.jobs.folder.prod
params:
  actions:
    GCloud:
      params:
        project_name: zebra-cicd
    Helm:
      params:
        namespace: kube-system
        chart_prefix: stable
        chart_name: traefik
    Kubectl:
      params:
        template_name: traefik
        namespace: kube-system
        chart_prefix: stable
        chart_name: traefik
`)

var testDrupipeHelmYaml = []byte(`---
from:
  - helm/blocks.yaml
  - helm/jobs.yaml
tags:
  - single
  - helm
pipeline_script_full: Jenkinsfile
params:
  actions:
    GCloud:
      params:
        cluster_name: main
        access_key_file_id: GCLOUD_ACCESS_KEY
  jobs:
    folder:
      helm:
        general:
          install:
            params:
              jobs:
                status:
                  from: .params.jobs.common.helm.status
                install:
                  from:
                    - .params.jobs.common
                  pipeline:
                    from: .params.pipelines.helm.install
                destroy:
                  from: .params.jobs.common.helm.destroy
    common:
      bump_stable:
        params:
          pipeline:
            from: .params.pipelines.bump_stable
      helm:
        install:
          params:
            pipeline:
              from: .params.pipelines.helm.install
        status:
          params:
            pipeline:
              from: .params.pipelines.helm.status
        destroy:
          params:
            pipeline:
              from: .params.pipelines.helm.destroy
    gitlab:
      params:
        name: 'jobs.gitlab'
      mr:
        params:
          branch: "${GIT_COMMIT}"
      webhooks:
        push:
          params:
            webhooks:
              - push_events: true
        mr:
          params:
            webhooks:
              - push_events: false
                merge_requests_events: true
      triggers:
        push:
          params:
            triggers:
              gitlabPush:
                buildOnPushEvents: true
                buildOnMergeRequestEvents: false
                enableCiSkip: true
          develop:
            params:
              triggers:
                gitlabPush:
                  includeBranches: ['develop']
          master:
            params:
              triggers:
                gitlabPush:
                  includeBranches: ['master']
        mr:
          params:
            triggers:
              gitlabPush:
                buildOnPushEvents: false
                rebuildOpenMergeRequest: 'source'
                includeBranches: ['master']
                enableCiSkip: true
  pipelines:
    params:
      name: default
    bump_stable:
      params:
        pods:
        - from: .params.pods.default
          containers:
          - from: .params.containers.common
            blocks:
            - from: .params.blocks.bump-stable
    helm:
      status:
        params:
          pods:
          - from: .params.pods.helm
            containers:
            - from: .params.containers.helm.status
            - from: .params.containers.kubectl.status
      install:
        params:
          pods:
          - from: .params.pods.helm
            containers:
            - from: .params.containers.helm
              blocks:
              - from: .params.blocks.helm.${context.container_types.helm.apply.type}.apply
              - from: .params.blocks.helm.status
      destroy:
        params:
          pods:
          - from: .params.pods.helm.destroy
  pods:
    helm:
      params:
        pre_containers:
        - from: .params.containers.gcloud
          blocks:
          - from: .params.blocks.gcloud.auth
      destroy:
        params:
          containers:
          - from: .params.containers.helm.destroy
  containers:
    gcloud:
      params:
        name: gcloud
        image: google/cloud-sdk:alpine
      auth:
        params:
          blocks:
            - from: .params.blocks.gcloud.auth
    kubectl:
      params:
        name: kubectl
        image: lachlanevenson/k8s-kubectl:v1.8.2
      status:
        params:
          blocks:
            - from: .params.blocks.kubectl.status
    helm:
      params:
        name: kubectl
        image: lachlanevenson/k8s-helm:v2.7.2
      status:
        params:
          blocks:
            - from: .params.blocks.helm.status
      destroy:
        params:
          blocks:
            - from: .params.blocks.helm.destroy
`)

var testDrupipeHelmBlocksYaml = []byte(`---
params:
  blocks:
    kubectl:
      rescale:
        params:
          actions:
          - from: .params.actions.Kubectl.scale_down_up
      status:
        params:
          actions:
          - from: .params.actions.Kubectl.get_pods
      pod-logs:
        params:
          actions:
          - from: .params.actions.Kubectl.get_pod_name
          - from: .params.actions.Kubectl.get_pod_logs
            pod_name: "${context.results.action.Kubectl_get_pod_name.stdout}"
    healthcheck:
      wait-http-200:
        params:
          actions:
          - from: .params.actions.HealthCheck.wait_http_ok
    gcloud:
      auth:
        params:
          actions:
          - from: .params.actions.GCloud.auth
    helm:
      params:
        pre_actions:
        - from: .params.actions.Helm.init
      status:
        params:
          actions:
          - from: .params.actions.Helm.status
      apply:
        params:
          actions:
          - from: .params.actions.Helm.${context.container_types.helm.apply.type}.apply
      destroy:
        params:
          actions:
          - from: .params.actions.Helm.delete
    gitlab:
      accept-mr:
        params:
          actions:
          - from: .params.actions.Gitlab.acceptMR
            message: All tests passed.
    bump-stable:
      params:
        actions:
        - from: .params.actions.Docman.bumpStable
    get-stable-version:
      params:
        actions:
        - from: .params.actions.Docman.getStable
          dir: stable_version
`)

var testDrupipeHelmJobsYaml = []byte(`---
jobs:
  dev:
    from:
      - .params.jobs.folder.helm.install-jenkins
  preprod:
    from:
      - .params.jobs.folder.helm.install-jenkins
  prod:
    from:
      - .params.jobs.folder.helm.install-jenkins
  merge-requests:
    jobs:
      install:
        pipeline:
          from: .params.pipelines.helm.install_jenkins
`)

var testDrupipeV3ActionsYaml = []byte(`---
`)

