package uniconf_test

import (
	"bytes"
	"github.com/aroq/uniconf/uniconf"
	"github.com/aroq/uniconf/unitool"
	"github.com/juju/testing/checkers"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// PrepareTest provides config values.
func PrepareTest() {
	uniconf.New()

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
	uniconf.AddSource(uniconf.NewSourceConfigMap("root", map[string]interface{}{
		"configMap": map[string]interface{}{
			"root": rootConfig(),
		},
	}))
	uniconf.SetRootSource("root")

	uniconf.AddSource(uniconf.NewSourceConfigMap("project", map[string]interface{}{
		"configMap": map[string]interface{}{
			"root": testHelmProjectYaml,
		},
	}))
	uniconf.AddSource(uniconf.NewSourceConfigMap("drupipe", map[string]interface{}{
		"configMap": map[string]interface{}{
			"helm":        testDrupipeHelmYaml,
			"helm/blocks": testDrupipeHelmBlocksYaml,
			"helm/jobs":   testDrupipeHelmJobsYaml,
			"v3":          testDrupipeV3Yaml,
			"v3/actions":  testDrupipeV3ActionsYaml,
		},
	}))
}

// PrepareFromHierarchyTest provides config values.
func PrepareFromHierarchyTest() {
	uniconf.New()

	// Set environment variables.
	var envVars = map[string][]byte{
		"UNICONF": []byte(`{
			"log_level": "INFO"
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
				"env:UNICONF",
			},
		}
	}

	// Provide sources & config entities.
	uniconf.AddSource(uniconf.NewSourceConfigMap("root", map[string]interface{}{
		"configMap": map[string]interface{}{
			"root": rootConfig(),
		},
	}))
	uniconf.SetRootSource("root")

	uniconf.AddSource(uniconf.NewSourceConfigMap("project", map[string]interface{}{
		"configMap": map[string]interface{}{
			"root": testFromHierarchyYaml,
		},
	}))
}

// TestLoad tests config load & basic functions.
func TestLoad(t *testing.T) {
	PrepareTest()

	uniconf.AddPhase(&uniconf.Phase{
		Name: "config",
		Phases: []*uniconf.Phase{
			{
				Name:     "load",
				Callback: uniconf.Load,
			},
			//{
			//	Name:     "print",
			//	Callback: uniconf.PrintConfig,
			//	//Args: []interface{}{
			//	//	"jobs.prod.jobs.install",
			//	//},
			//},
		},
	})

	uniconf.Execute()

	t.Run("u.config defined", func(t *testing.T) {
		assert.NotEqual(t, uniconf.Config(), nil, "no config defined")
	})
	t.Run("log_level", func(t *testing.T) {
		assert.Contains(t, uniconf.Config(), "log_level", "no 'log_level' key in config")
		assert.Equal(t, uniconf.Config()["log_level"], "DEBUG", "log_level should equal 'DEBUG'")

		//t.Run("history==env:UNICONF", func(t *testing.T) {
		//	if _, ok := uniconf.Config().history["log_level"].(map[string]interface{})["load"].(map[string]interface{})["env:UNICONF"]; !ok {
		//		t.Errorf("Uniconf history failed env:UNICONF")
		//	}
		//})
	})
	t.Run("Collect(params.jobs.common.helm.install).pipeline.from = '.params.pipelines.helm.install'", func(t *testing.T) {
		path := "params.jobs.common.helm.install"
		params, _ := unitool.DeepCollectParams(uniconf.Config(), path, "params")
		assert.Equal(t, unitool.SearchMapWithPathStringPrefixes(params, "pipeline.from"), ".params.pipelines.helm.install", "Deep key search failed: %s", path)
	})

	t.Run("Compare Load() result", func(t *testing.T) {
		i1 := uniconf.Config()
		i2, _ := unitool.UnmarshalYaml(testLoadResult)
		result, err := AreEqualInterfaces(i1, i2)
		assert.Equal(t, result, true, "Compare Load() result failed: %v", err)
	})
}

// TestLoad tests config load & basic functions.
func TestLoadWithFlattenConfig(t *testing.T) {
	PrepareTest()

	uniconf.AddPhase(&uniconf.Phase{
		Name: "config",
		Phases: []*uniconf.Phase{
			{
				Name:     "load",
				Callback: uniconf.Load,
			},
			{
				Name:     "flatten_config",
				Callback: uniconf.FlattenConfig,
			},
		},
	})

	uniconf.Execute()

	t.Run("InterpolateString", func(t *testing.T) {
		t.Run("${log_level}==DEBUG", func(t *testing.T) {
			result := uniconf.InterpolateString("${log_level}", nil)
			if result != "DEBUG" {
				t.Errorf("Interpolate string failed: expected value: 'master', real value: %v", result)
			}
		})
		t.Run("${deepGet(log_level)}==DEBUG", func(t *testing.T) {
			result := uniconf.InterpolateString("${deepGet(\"log_level\")}", nil)
			if result != "DEBUG" {
				t.Errorf("Interpolate string with initial deepGet() failed: expected value: 'master', real value: %v", result)
			}
		})
	})
}

// TestLoadFromProcess tests config load & basic functions.
func TestLoadFromProcess(t *testing.T) {
	PrepareTest()

	var job interface{}
	uniconf.AddPhase(&uniconf.Phase{
		Name: "config",
		Phases: []*uniconf.Phase{
			{
				Name:     "load",
				Callback: uniconf.Load,
			},
			{
				Name:     "flatten_config",
				Callback: uniconf.FlattenConfig,
			},
			{
				Name:     "get_job",
				Callback: uniconf.ProcessContext,
				Args: []interface{}{
					"job",
					"prod.install",
				},
				Result: &job,
			},
		},
	})

	uniconf.Execute()

	t.Run("environment", func(t *testing.T) {
		assert.Contains(t, uniconf.Config(), "environment", "no 'environment' key in config")
		assert.Equal(t, uniconf.Config()["environment"], "prod", "environment should equal 'prod'")
	})

	t.Run("Compare processed job result", func(t *testing.T) {
		i1, _ := unitool.UnmarshalYaml([]byte(unitool.MarshallYaml(job)))
		//log.Println(unitool.MarshallYaml(job))
		i2, _ := unitool.UnmarshalYaml(testHelmProdInstallJobResult)
		result, err := AreEqualInterfaces(i1, i2)
		assert.Equal(t, result, true, "Compare processed job result failed: %v", err)
	})
}

// TestLoadFromHierarchyProcess tests config load & basic functions.
func TestLoadFromHierarchyProcess(t *testing.T) {
	PrepareFromHierarchyTest()

	uniconf.AddPhase(&uniconf.Phase{
		Name: "config",
		Phases: []*uniconf.Phase{
			{
				Name:     "load",
				Callback: uniconf.Load,
			},
			{
				Name:     "flatten_config",
				Callback: uniconf.FlattenConfig,
			},
		},
	})

	uniconf.AddPhase(&uniconf.Phase{
		Name: "process",
		Phases: []*uniconf.Phase{
			{
				Name:     "process",
				Callback: uniconf.ProcessKeys,
				Args: []interface{}{
					"projects",
					"",
					[]*uniconf.Processor{
						{
							Callback:    uniconf.FromProcess,
							IncludeKeys: []string{uniconf.IncludeListElementName},
						},
					},
				},
			},
			{
				Name:     "print",
				Callback: uniconf.PrintConfig,
			},
		},
	})

	uniconf.Execute()

	//t.Run("environment", func(t *testing.T) {
	//	assert.Contains(t, uniconf.Config(), "environment", "no 'environment' key in config")
	//	assert.Equal(t, uniconf.Config()["environment"], "prod", "environment should equal 'prod'")
	//})
	//
	//t.Run("Compare processed job result", func(t *testing.T) {
	//	i1, _ := unitool.UnmarshalYaml([]byte(unitool.MarshallYaml(job)))
	//	i2, _ := unitool.UnmarshalYaml(testHelmProdInstallJobResult)
	//	result, err := AreEqualInterfaces(i1, i2)
	//	assert.Equal(t, result, true, "Compare processed job result failed: %v", err)
	//})
}

func AreEqualInterfaces(i1, i2 interface{}) (bool, error) {
	return checkers.DeepEqual(i1, i2)
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
  dev:
    from:
      - .params.jobs.folder.dev
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
        name: helm
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
`)

var testDrupipeV3Yaml = []byte(`---
entities:
  job:
    retrieve_handler: DeepCollectChildren
    children_key: jobs
    context_name: job
    processors:
      - from_processor
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
params:
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

var testLoadResult = []byte(`---
entities:
  job:
    retrieve_handler: DeepCollectChildren
    children_key: jobs
    context_name: job
    processors:
      - from_processor
container_types:
  helm:
    apply:
      type: chart
from_processed:
- helm/blocks
- helm/jobs
- drupipe:helm
- drupipe:v3/actions
- env:UNICONF
- project:root
- env:UNICONF_TEST_MULTIPART_ENVVAR
jobs:
  dev:
    from:
    - .params.jobs.folder.dev
    - .params.jobs.folder.helm.general.install
    - .params.jobs.folder.dev
    jobs:
      install:
        from:
        - .params.jobs.gitlab.triggers.push.develop
        - .params.jobs.gitlab.webhooks.push
  merge-requests:
    from: .params.jobs.folder.mr
    jobs:
      destroy:
        from: .params.jobs.common.helm.destroy
      install:
        from:
        - .params.jobs.common
        - .params.jobs.gitlab.mr
        - .params.jobs.gitlab.triggers.mr
        - .params.jobs.gitlab.webhooks.mr
        pipeline:
          final_pods:
          - containers:
            - from: .params.containers.helm.destroy
            from: .params.pods.default
          post_pods:
          - containers:
            - blocks:
              - from: .params.blocks.gitlab.accept-mr
              from: .params.containers.common
            from: .params.pods.master
      status:
        from: .params.jobs.common.helm.status
  preprod:
    from:
    - .params.jobs.folder.preprod
    - .params.jobs.folder.helm.general.install
  prod:
    from:
    - .params.jobs.folder.prod
    - .params.jobs.folder.helm.general.install
    - .params.jobs.folder.prod
log_level: DEBUG
params:
  actions:
    GCloud:
      params:
        access_key_file_id: GCLOUD_ACCESS_KEY
        cluster_name: main
        project_name: zebra-cicd
    Helm:
      params:
        chart_name: traefik
        chart_prefix: stable
        namespace: kube-system
    Kubectl:
      params:
        chart_name: traefik
        chart_prefix: stable
        namespace: kube-system
        template_name: traefik
    params:
      action_timeout: 120
      dump_result: true
      fallback_class_name: BaseShellAction
      hooks:
      - params
      result_post_process:
        result:
          destination: ${action.params.store_result_key}
          source: result
          type: result
      return_stdout: false
      shell_bash_login: true
      store_action_params: true
      store_action_params_key: actions.${action.name}_${action.methodName}
      store_result: true
      store_result_key: context.results.action.${action.name}_${action.methodName}
    processors:
      from:
        mode: execute
  blocks:
    bump-stable:
      params:
        actions:
        - from: .params.actions.Docman.bumpStable
    gcloud:
      auth:
        params:
          actions:
          - from: .params.actions.GCloud.auth
    get-stable-version:
      params:
        actions:
        - dir: stable_version
          from: .params.actions.Docman.getStable
    gitlab:
      accept-mr:
        params:
          actions:
          - from: .params.actions.Gitlab.acceptMR
            message: All tests passed.
    healthcheck:
      wait-http-200:
        params:
          actions:
          - from: .params.actions.HealthCheck.wait_http_ok
    helm:
      apply:
        params:
          actions:
          - from: .params.actions.Helm.${context.container_types.helm.apply.type}.apply
      destroy:
        params:
          actions:
          - from: .params.actions.Helm.delete
      params:
        pre_actions:
        - from: .params.actions.Helm.init
      status:
        params:
          actions:
          - from: .params.actions.Helm.status
    kubectl:
      pod-logs:
        params:
          actions:
          - from: .params.actions.Kubectl.get_pod_name
          - from: .params.actions.Kubectl.get_pod_logs
            pod_name: ${context.results.action.Kubectl_get_pod_name.stdout}
      rescale:
        params:
          actions:
          - from: .params.actions.Kubectl.scale_down_up
      status:
        params:
          actions:
          - from: .params.actions.Kubectl.get_pods
  containers:
    common:
      params:
        image: michaeltigr/zebra-build-php-drush-docman:0.0.87
        name: common
    gcloud:
      auth:
        params:
          blocks:
          - from: .params.blocks.gcloud.auth
      params:
        image: google/cloud-sdk:alpine
        name: gcloud
    helm:
      destroy:
        params:
          blocks:
          - from: .params.blocks.helm.destroy
      params:
        image: lachlanevenson/k8s-helm:v2.7.2
        name: helm
      status:
        params:
          blocks:
          - from: .params.blocks.helm.status
    kubectl:
      params:
        image: lachlanevenson/k8s-kubectl:v1.8.2
        name: kubectl
      status:
        params:
          blocks:
          - from: .params.blocks.kubectl.status
    none:
      params:
        name: none
    options:
      ssh_tunnel:
        params:
          pre_blocks:
          - actions:
            - from: .params.actions.Ssh.tunnel
    params:
      execute: true
      k8s:
        alwaysPullImage: true
        command: cat
        resourceLimitCpu: 500m
        resourceLimitMemory: 1000Mi
        resourceRequestCpu: 50m
        resourceRequestMemory: 200Mi
        ttyEnabled: true
  jobs:
    common:
      bump_stable:
        params:
          pipeline:
            from: .params.pipelines.bump_stable
      helm:
        destroy:
          params:
            pipeline:
              from: .params.pipelines.helm.destroy
        install:
          params:
            pipeline:
              from: .params.pipelines.helm.install
        status:
          params:
            pipeline:
              from: .params.pipelines.helm.status
      params:
        type: common
    folder:
      dev:
        params:
          branch: develop
          context:
            environment: dev
      helm:
        general:
          install:
            params:
              jobs:
                destroy:
                  from: .params.jobs.common.helm.destroy
                install:
                  from:
                  - .params.jobs.common
                  pipeline:
                    from: .params.pipelines.helm.install
                status:
                  from: .params.jobs.common.helm.status
      mr:
        params:
          context:
            environment: mr
      params:
        type: folder
      preprod:
        params:
          branch: master
          context:
            environment: preprod
      prod:
        params:
          branch: master
          context:
            environment: prod
    gitlab:
      mr:
        params:
          branch: ${GIT_COMMIT}
      params:
        name: jobs.gitlab
      triggers:
        mr:
          params:
            triggers:
              gitlabPush:
                buildOnPushEvents: false
                enableCiSkip: true
                includeBranches:
                - master
                rebuildOpenMergeRequest: source
        push:
          develop:
            params:
              triggers:
                gitlabPush:
                  includeBranches:
                  - develop
          master:
            params:
              triggers:
                gitlabPush:
                  includeBranches:
                  - master
          params:
            triggers:
              gitlabPush:
                buildOnMergeRequestEvents: false
                buildOnPushEvents: true
                enableCiSkip: true
      webhooks:
        mr:
          params:
            webhooks:
            - merge_requests_events: true
              push_events: false
        push:
          params:
            webhooks:
            - push_events: true
  options:
    actions:
      pre:
        ssh_tunnel:
          params:
            pre_blocks:
            - actions:
              - from: .params.actions.Ssh.tunnel
    containers:
      build:
        tools:
          params:
            image: michaeltigr/zebra-build-php-drush-docman-tools:0.0.87
      k8s:
        large:
          params:
            k8s:
              resourceLimitCpu: 2000m
              resourceLimitMemory: 2000Mi
              resourceRequestCpu: 500m
              resourceRequestMemory: 1000Mi
        medium:
          params:
            k8s:
              resourceLimitCpu: 1000m
              resourceLimitMemory: 1500Mi
              resourceRequestCpu: 100m
              resourceRequestMemory: 500Mi
        small:
          params:
            k8s:
              resourceLimitCpu: 500m
              resourceLimitMemory: 1000Mi
              resourceRequestCpu: 50m
              resourceRequestMemory: 250Mi
  pipelines:
    bump_stable:
      params:
        pods:
        - containers:
          - blocks:
            - from: .params.blocks.bump-stable
            from: .params.containers.common
          from: .params.pods.default
    helm:
      destroy:
        params:
          pods:
          - from: .params.pods.helm.destroy
      install:
        params:
          pods:
          - containers:
            - blocks:
              - from: .params.blocks.helm.${context.container_types.helm.apply.type}.apply
              - from: .params.blocks.helm.status
              from: .params.containers.helm
            from: .params.pods.helm
      status:
        params:
          pods:
          - containers:
            - from: .params.containers.helm.status
            - from: .params.containers.kubectl.status
            from: .params.pods.helm
    params:
      name: default
  pods:
    default:
      params:
        name: default
    helm:
      destroy:
        params:
          containers:
          - from: .params.containers.helm.destroy
      params:
        pre_containers:
        - blocks:
          - from: .params.blocks.gcloud.auth
          from: .params.containers.gcloud
    master:
      params:
        containerized: false
        name: master
    params:
      containerized: true
      unipipe_retrieve_config: true
  processors:
    from:
      mode: config
sources:
  env:
    type: env
tags:
- single
- helm
`)

var testHelmProdInstallJobResult = []byte(`---
branch: master
context:
  environment: prod
from:
- .params.jobs.folder.prod
from_processed:
- .params.jobs.folder.prod
- .params.jobs.folder.helm.general.install
- .params.jobs.common
pipeline:
  from_processed:
  - .params.pipelines.helm.install
  name: default
  pods:
  - containerized: true
    containers:
    - blocks:
      - from_processed:
        - .params.blocks.helm.${context.container_types.helm.apply.type}.apply (.params.blocks.helm.chart.apply)
        pre_actions:
        - from: .params.actions.Helm.init
      - actions:
        - from: .params.actions.Helm.status
        from_processed:
        - .params.blocks.helm.status
        pre_actions:
        - from: .params.actions.Helm.init
      execute: true
      from_processed:
      - .params.containers.helm
      image: lachlanevenson/k8s-helm:v2.7.2
      k8s:
        alwaysPullImage: true
        command: cat
        resourceLimitCpu: 500m
        resourceLimitMemory: 1000Mi
        resourceRequestCpu: 50m
        resourceRequestMemory: 200Mi
        ttyEnabled: true
      name: helm
    from_processed:
    - .params.pods.helm
    pre_containers:
    - blocks:
      - actions:
        - from: .params.actions.GCloud.auth
        from_processed:
        - .params.blocks.gcloud.auth
      execute: true
      from_processed:
      - .params.containers.gcloud
      image: google/cloud-sdk:alpine
      k8s:
        alwaysPullImage: true
        command: cat
        resourceLimitCpu: 500m
        resourceLimitMemory: 1000Mi
        resourceRequestCpu: 50m
        resourceRequestMemory: 200Mi
        ttyEnabled: true
      name: gcloud
    unipipe_retrieve_config: true
type: common
`)

var testFromHierarchyYaml = []byte(`---
defaults:
  mothership_workflow_deploy_direct: &mothership_workflow_deploy_direct
  - Code is delivered directly to environment (without artifact build stage) to reduce deploy time.
  mothership_workflow_deploy_direct_params: &mothership_workflow_deploy_direct_params
    params:
      labels:
      - deploy-direct
  mothership_workflow_operations_drush: &mothership_workflow_operations_drush
  - Operations are performed with drush.
  mothership_workflow_operation_drush_params: &mothership_workflow_operations_drush_params
    params:
      components:
      - drush
mothership:
  project:
    type1:
      params:
        from:
        - .mothership.project.dev.direct.composer
        params:
          labels:
          - mothership.project.type.cicd.common.single.type1
    dev:
      direct:
        composer:
          params:
            from:
            - .mothership.project.workflow.dev.deploy.direct
            - .mothership.project.workflow.dev.operations.drush
    workflow:
      dev:
        deploy:
          direct:
            params:
              <<: *mothership_workflow_deploy_direct_params
              workflow:
                dev:
                  deploy: *mothership_workflow_deploy_direct
        operations:
          drush:
            params:
              <<: *mothership_workflow_operations_drush_params
              workflow:
                dev:
                  operations: *mothership_workflow_operations_drush
projects:
  Test:
    from:
    - .mothership.project.type1
`)
