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
				"project:root",
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
			"root": testHelmProjectYaml,
		},
	}))

	AddSource(NewSourceConfigMap("drupipe", map[string]interface{}{
		"configMap": map[string]interface{}{
			"helm":        testDrupipeHelmYaml,
			"helm/blocks": testDrupipeHelmBlocksYaml,
			"helm/jobs":   testDrupipeHelmJobsYaml,
			"v3":  testDrupipeV3Yaml,
			"v3/actions":  testDrupipeV3ActionsYaml,
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
func TestLoadFromProcess(t *testing.T) {
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
			{
				Name:     "flatten_config",
				Callback: FlattenConfig,
			},
			{
				Name:     "process",
				Callback: ProcessKeys,
				Args: []interface{}{
					"prod.install",
					"jobs",
					[]*Processor{
						{
							Callback:    FromProcess,
							IncludeKeys: []string{"from"},
						},
					},
				},
			},
			{
				Name:     "print",
				Callback: PrintConfig,
				Args: []interface{}{
					"jobs.prod.jobs.install",
				},
			},
		},
	})

	SetContexts("jobs.prod.jobs.install")

	Execute()

	job, _ := unitool.CollectInvertedKeyParamsFromJsonPath(u.config, "prod.install", "jobs")

	path := "pipeline.name"
	value := "default"
	result := unitool.SearchMapWithPathStringPrefixes(job, path)
	if result != value {
		t.Errorf("Deep key search failed: %s, expected value: %v, real value: %v", path, value, result)
	}
}

// Test config entities.
var testHelmProjectYaml = []byte(`---
from:
  - drupipe:helm
  - drupipe:v3/actions
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
  - helm/blocks
  - helm/jobs
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
      - .params.jobs.folder.dev
      - .params.jobs.folder.helm.general.install
    jobs:
      install:
        from:
          # - .params.jobs.common.helm.install
          - .params.jobs.gitlab.triggers.push.develop
          - .params.jobs.gitlab.webhooks.push

  preprod:
    from:
      - .params.jobs.folder.preprod
      - .params.jobs.folder.helm.general.install

  prod:
    from:
      - .params.jobs.folder.prod
      - .params.jobs.folder.helm.general.install

  merge-requests:
    from: .params.jobs.folder.mr
    jobs:
      status:
        from: .params.jobs.common.helm.status

      install:
        from:
          - .params.jobs.common
          - .params.jobs.gitlab.mr
          - .params.jobs.gitlab.triggers.mr
          - .params.jobs.gitlab.webhooks.mr
        pipeline:
          post_pods:
            - from: .params.pods.master
              containers:
                - from: .params.containers.common
                  blocks:
                    - from: .params.blocks.gitlab.accept-mr
          final_pods:
            # TODO: add clean slaves kubectl command here.
            - from: .params.pods.default
              containers:
                - from: .params.containers.helm.destroy

      destroy:
        from: .params.jobs.common.helm.destroy

  # stable-bump:
    # from:
      # - .params.jobs.common.bump_stable
      # - .params.jobs.gitlab.triggers.push.master
      # - .params.jobs.gitlab.webhooks.push
`)

var testDrupipeV3Yaml = []byte(`---
params:
  actions:
    processors:
      from:
        # defines in which mode 'from' should be processed.
        mode: execute
    params:
      action_timeout: 120
      store_result: true
      dump_result: true
      store_action_params: true
      store_result_key: context.results.action.${action.name}_${action.methodName}
      hooks: ['params']
      # Result post process
      result_post_process:
        result:
          type: result
          source: result
          destination: ${action.params.store_result_key}
      store_action_params_key: actions.${action.name}_${action.methodName}
      shell_bash_login: true
      return_stdout: false
      # To be used when specific action's class is not exists.
      fallback_class_name: BaseShellAction
`)

var testDrupipeV3ActionsYaml = []byte(`---
config_version: 3

log_level: INFO

log_levels:
  TRACE:
    weight: 10
    color: cyan
  DEBUG:
    weight: 20
    color: yellow
  INFO:
    weight: 30
    color: green
  WARNING:
    weight: 40
    color: red
  ERROR:
    weight: 50
    color: magenta

uniconf:
  keys:
    include: scenarios
    sources: scenarioSources
    params: params
    jobs: jobs
    processors: processors
  dirs:
    sources: scenarios
  files:
    config_file_name: config.yaml
  include:
    prefix: .params.
    separator: '|'

environment: ''
debugEnabled: false
docrootDir: docroot
docmanDir: docman
projectConfigPath: .unipipe/config
projectConfigFile: docroot.config
containerMode: docker
configSeedType: docman
defaultDocmanImage: michaeltigr/zebra-build-php-drush-docman:0.0.87
logRotatorNumToKeep: 5
drupipeDockerArgs: --user root:root --net=host

processors:
  - className: DrupipeFromProcessor
    properties:
      include_key: from

config_providers_list:
  - env
  - mothership
  - project
  - job

config_providers:
  env:
    class_name: ConfigProviderEnv
  mothership:
    class_name: ConfigProviderMothership
  project:
    class_name: ConfigProviderProject
  job:
    class_name: ConfigProviderJob

params:

jobs:
  mothership:
    type: mothership
    pipeline:
      pods:
      - from: .params.pods.master
        containers:
        - from: .params.containers.common
          blocks:
          - actions:
            - from: .params.actions.JobDslSeed.perform
              dsl_params:
                lookupStrategy: JENKINS_ROOT
                jobsPattern: ['.unipipe/library/jobdsl/job_dsl_mothership.groovy']
                override: true
                removedJobAction: DELETE
                removedViewAction: DELETE
                additionalClasspath: ['.unipipe/library/src']

  seed:
    type: seed
    pipeline:
      pods:
      - from: .params.pods.default
        containers:
        - from: .params.containers.common
          blocks:
          - actions:
            - from: .params.actions.JobDslSeed.info
      - from: .params.pods.master
        containers:
        - from: .params.containers.common
          blocks:
          - actions:
            - from: .params.actions.JobDslSeed.prepare
            - from: .params.actions.JobDslSeed.perform

params:
  pipeline:
    scripts_library:
      url: https://github.com/aroq/drupipe.git
      ref: master
      type: branch

  block:
    nodeName: default
    # TODO: remove it after configs update.
    dockerImage: aroq/drudock:1.4.0

  processors:
      from:
        # defines in which mode 'from' should be processed.
        mode: config

  jobs:
    folder:
      params:
        type: folder
      dev:
        params:
          branch: develop
          # Context params will be merged into main pipeline context.
          context:
            environment: dev
      preprod:
        params:
          branch: master
          # Context params will be merged into main pipeline context.
          context:
            environment: preprod
      prod:
        params:
          branch: master
          # Context params will be merged into main pipeline context.
          context:
            environment: prod
      mr:
        params:
          # Context params will be merged into main pipeline context.
          context:
            environment: mr
    common:
      params:
        type: common

  pipelines:
    params:
      name: default

  pods:
    params:
      unipipe_retrieve_config: true
      containerized: true

    master:
      params:
        containerized: false
        name: master

    default:
      params:
        name: default

  containers:
    params:
      execute: true
      k8s:
        ttyEnabled: true
        command: cat
        resourceRequestCpu: 50m
        resourceLimitCpu: 500m
        resourceRequestMemory: 200Mi
        resourceLimitMemory: 1000Mi
        alwaysPullImage: true
    none:
      params:
        name: none
    common:
      params:
        name: common
        image: michaeltigr/zebra-build-php-drush-docman:0.0.87
    options:
      ssh_tunnel:
        params:
          pre_blocks:
            - actions:
              - from: .params.actions.Ssh.tunnel
  options:
    containers:
      build:
        tools:
          params:
            image: michaeltigr/zebra-build-php-drush-docman-tools:0.0.87
      k8s:
        small:
          params:
            k8s:
              resourceRequestCpu: 50m
              resourceLimitCpu: 500m
              resourceRequestMemory: 250Mi
              resourceLimitMemory: 1000Mi
        medium:
          params:
            k8s:
              resourceRequestCpu: 100m
              resourceLimitCpu: 1000m
              resourceRequestMemory: 500Mi
              resourceLimitMemory: 1500Mi
        large:
          params:
            k8s:
              resourceRequestCpu: 500m
              resourceLimitCpu: 2000m
              resourceRequestMemory: 1000Mi
              resourceLimitMemory: 2000Mi
    actions:
      pre:
        ssh_tunnel:
          params:
            pre_blocks:
              - actions:
                - from: .params.actions.Ssh.tunnel
`)

