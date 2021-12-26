package config_test

import (
	"testing"

	"github.com/sanservices/kit/config"
)

type testConfig struct {
	Input  config.Env
	Output *config.General
	Panic  bool
}

var testSuiteConfig = map[string]testConfig{
	"Successful": {
		config.Dev,
		&config.General{
			config.Info{
				Endpoint: "github.com",
			},
		}, false},
	"FileInexistant": {config.Prod, &config.General{}, true},
	"FileNotJSON":    {config.Test, &config.General{}, true},
}

func recoverer(t *testing.T, name string, current testConfig) {
	defer func(t *testing.T, name string, current testConfig) {
		if r := recover(); r != nil {
			if !current.Panic {
				t.Errorf("Test %s not expected to face panicked", name)
			}
		}
	}(t, name, current)
	result := config.Read(current.Input)
	if current.Panic {
		t.Errorf("Test %s expected to face panicked", name)
	}
	if result.Info.Endpoint != current.Output.Info.Endpoint {
		t.Errorf("Test %s expected to be %v not equla to %v", name, result.Info.Endpoint, current.Output.Info.Endpoint)
	}
}

func TestRead(t *testing.T) {
	for k, v := range testSuiteConfig {
		recoverer(t, k, v)
	}
}
