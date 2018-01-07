package unitool

import (
	"testing"
)

func TestMerge(t *testing.T) {
	src := map[string]interface{}{
		"key1": map[string]interface{}{
			"key1_subkey1": "key1_subkey1_value",
			"key1_subkey2": "key1_subkey2_value",
		},
		"key2": map[string]interface{}{
			"key2_subkey1": "key2_subkey1_value",
		},
	}

	dest := map[string]interface{}{
		"key1": map[string]interface{}{
			"key1_subkey2": "key1_subkey2_value_override",
			"key1_subkey3": "key1_subkey3_value",
		},
		"key2": map[string]interface{}{
			"key2_subkey1": "key2_subkey1_value",
		},
	}

	Merge(src, dest)

    // Test deep key merging.
	if src["key1"].(map[string]interface{})["key1_subkey2"] != "key1_subkey2_value_override" {
		t.Errorf("Deep key merging failed: %s", "key1_subkey2 != key1_subkey2_value_override")
	}

	// Test key merging.
	if src["key2"].(map[string]interface{})["key2_subkey1"] != "key2_subkey1_value" {
		t.Errorf("Deep key merging failed: %s", "key2_subkey1 != key2_subkey1_value")
	}
}

func TestUnmarshalYaml(t *testing.T) {
	yaml, err := UnmarshalYaml(yamlExample)

	if err != nil {
		t.Errorf("UnmarshalYaml err: %v", err)
	}

	if yaml["pipeline"].(map[string]interface{})["name"] != "helm.install_jenkins" {
		t.Errorf("Deep key retrieve failed: %s", "pipeline.name != helm.install_jenkins")
	}
}

func TestSearchMapWithPathStringPrefixes(t*testing.T) {
	src := map[string]interface{}{
		"key1": map[string]interface{}{
			"key1_subkey1": "key1_subkey1_value",
			"key1_subkey2": "key1_subkey2_value",
		},
		"key2": map[string]interface{}{
			"key2_subkey1": "key2_subkey1_value",
		},
	}
	path := "key1.key1_subkey1"
	value := "key1_subkey1_value"

	if SearchMapWithPathStringPrefixes(src, path) != value {
		t.Errorf("Deep key search failed: %s %v", path, value)
	}
}

func TestSearchMapWithPathStringPrefixesInYaml(t*testing.T) {
	src, err := UnmarshalYaml(yamlExample)
	if err != nil {
		t.Errorf("UnmarshalYaml err: %v", err)
	}

	path := "triggers.gitlabPush.buildOnPushEvents"
	value := true
	if SearchMapWithPathStringPrefixes(src, path) != value {
		t.Errorf("Deep key search failed: %s %v", path, value)
	}
}

func TestCollectKeyParamsFromJsonPath(t *testing.T) {
	src, err := UnmarshalYaml(yamlExample2)
	if err != nil {
		t.Errorf("UnmarshalYaml err: %v", err)
	}

	path := "params.jobs.dev"
	params, _ := CollectKeyParamsFromJsonPath(src, path, "params")
	//t.Logf("collectKeyParamsFromJsonPath result: %v", params)

	path = "jobs_param"
	value := false
	result := SearchMapWithPathStringPrefixes(params, path)
	if result != value {
		t.Errorf("Deep key search failed: %s, expected value: %v, real value: %v", path, value, result)
	}

	path = "jobs_dev_param"
	value = true
	result = SearchMapWithPathStringPrefixes(params, path)
	if result != value {
		t.Errorf("Deep key search failed: %s, expected value: %v, real value: %v", path, value, result)
	}

	path = "jobs"
	result = SearchMapWithPathStringPrefixes(params, path)
	if result != nil {
		t.Errorf("Deep key search failed: %s, expected value: nil, real value: %v", path, result)
	}
}

var yamlExample2 = []byte(`params:
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

var yamlExample = []byte(`type: common
branch: develop
context:
  environment: dev
from_processed: true
from_processed_mode: config
from_source: .params.jobs.common
pipeline:
  name: helm.install_jenkins
  pods:
  - containers:
    - blocks:
      - pre_actions:
        - from: .params.actions.Helm.init
        actions:
        - from: .params.actions.Helm.${context.container_types.helm.apply.type}.apply
        name: helm.chart_dir.apply
        from_processed: true
        from_processed_mode: config
        from_source: .params.blocks.helm.chart_dir.apply
      - pre_actions:
        - from: .params.actions.Helm.init
        actions:
        - from: .params.actions.Helm.status
        name: helm.status
        from_processed: true
        from_processed_mode: config
        from_source: .params.blocks.helm.status
      execute: true
      k8s:
        ttyEnabled: true
        command: cat
        resourceRequestCpu: 50m
        resourceLimitCpu: 500m
        resourceRequestMemory: 200Mi
        resourceLimitMemory: 1000Mi
        alwaysPullImage: true
      name: helm
      image: lachlanevenson/k8s-helm:v2.7.2
      from_processed: true
      from_processed_mode: config
      from_source: .params.containers.helm
    - blocks:
      - actions:
        - from: .params.actions.Kubectl.scale_down_up
        name: kubectl.rescale
        from_processed: true
        from_processed_mode: config
        from_source: .params.blocks.kubectl.rescale
      - actions:
        - from: .params.actions.Kubectl.get_pods
        post_actions:
        - from: .params.actions.Kubectl.get_address_${context.k8s.jenkins.service_type}
        name: kubectl.status
        from_processed: true
        from_processed_mode: config
        from_source: .params.blocks.kubectl.status
      execute: true
      k8s:
        ttyEnabled: true
        command: cat
        resourceRequestCpu: 50m
        resourceLimitCpu: 500m
        resourceRequestMemory: 200Mi
        resourceLimitMemory: 1000Mi
        alwaysPullImage: true
      name: kubectl
      image: lachlanevenson/k8s-kubectl:v1.8.2
      from_processed: true
      from_processed_mode: config
      from_source: .params.containers.kubectl
    - blocks:
      - actions:
        - from: .params.actions.HealthCheck.wait_http_ok
        name: healthcheck.wait-http-200
        from_processed: true
        from_processed_mode: config
        from_source: .params.blocks.healthcheck.wait-http-200
      execute: true
      k8s:
        ttyEnabled: true
        command: cat
        resourceRequestCpu: 50m
        resourceLimitCpu: 500m
        resourceRequestMemory: 200Mi
        resourceLimitMemory: 1000Mi
        alwaysPullImage: true
      name: common
      image: michaeltigr/zebra-build-php-drush-docman:0.0.87
      from_processed: true
      from_processed_mode: config
      from_source: .params.containers.common
    - blocks:
      - actions:
        - from: .params.actions.Kubectl.get_pod_name
        - from: .params.actions.Kubectl.get_pod_logs
          pod_name: ${context.results.action.Kubectl_get_pod_name.stdout}
        name: kubectl.pod-logs
        from_processed: true
        from_processed_mode: config
        from_source: .params.blocks.kubectl.pod-logs
      - actions:
        - from: .params.actions.Kubectl.get_pod_name
        - from: .params.actions.Kubectl.copy_from_pod
          pod_name: ${context.results.action.Kubectl_get_pod_name.stdout}
          source_file_name: /var/jenkins_home/zebra/user_token
          destination: jenkins_user_token
          result_post_process:
            jenkins_user_token_file:
              type: result
              source: params.destination
              destination: context.jenkins.user_token_file
        - from: .params.actions.Shell.execute
          shellCommand: cat ${context.jenkins.user_token_file}
        name: kubectl.copy-file
        from_processed: true
        from_processed_mode: config
        from_source: .params.blocks.kubectl.copy-file
      execute: true
      k8s:
        ttyEnabled: true
        command: cat
        resourceRequestCpu: 50m
        resourceLimitCpu: 500m
        resourceRequestMemory: 200Mi
        resourceLimitMemory: 1000Mi
        alwaysPullImage: true
      name: kubectl
      image: lachlanevenson/k8s-kubectl:v1.8.2
      from_processed: true
      from_processed_mode: config
      from_source: .params.containers.kubectl
    - blocks:
      - actions:
        - from: .params.actions.Jenkins.build
          params: null
          jenkins_address_host: ${context.k8s.address}
          jenkins_user_token_file: ${context.jenkins.user_token_file}
          jobName: mothership
          args: -s
          user: ${context.jenkins.username}
        - from: .params.actions.Jenkins.seedTest
          jenkins_address_host: ${context.k8s.address}
          jenkins_user_token_file: ${context.jenkins.user_token_file}
          args: -s
          user: ${context.jenkins.username}
        name: jenkins.test
        from_processed: true
        from_processed_mode: config
        from_source: .params.blocks.jenkins.test
      execute: true
      k8s:
        ttyEnabled: true
        command: cat
        resourceRequestCpu: 50m
        resourceLimitCpu: 500m
        resourceRequestMemory: 200Mi
        resourceLimitMemory: 1000Mi
        alwaysPullImage: true
      name: jenkins-cli
      image: michaeltigr/zebra-jenkins-cli:2.73.3
      from_processed: true
      from_processed_mode: config
      from_source: .params.containers.jenkins-cli
    unipipe_retrieve_config: true
    containerized: true
    name: default
    from_processed: true
    from_processed_mode: config
    from_source: .params.pods.default
  from_processed: true
  from_processed_mode: config
  from_source: .params.pipelines.helm.install_jenkins
name: jobs.gitlab
triggers:
  gitlabPush:
    test: true
    buildOnPushEvents: true
    buildOnMergeRequestEvents: false
    enableCiSkip: true
    includeBranches:
    - develop
webhooks:
- push_events: true
`)
