/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package options contains flags for initializing an autoscaler.
package options

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
)

// AutoScalerConfig configures and runs an autoscaler server
type AutoScalerConfig struct {
	Target            string
	ConfigMap         string
	Namespace         string
	DefaultParams     configMapData
	PollPeriodSeconds int
	PrintVer          bool
	NodeLabels        string
	MaxSyncFailures   int
}

// NewAutoScalerConfig returns a Autoscaler config
func NewAutoScalerConfig() *AutoScalerConfig {
	return &AutoScalerConfig{
		Namespace:         os.Getenv("MY_POD_NAMESPACE"),
		PollPeriodSeconds: 10,
		PrintVer:          false,
	}
}

// ValidateFlags validates whether flags are set up correctly
func (c *AutoScalerConfig) ValidateFlags() error {
	var errorsFound bool
	c.Target = strings.ToLower(c.Target)
	if !isTargetFormatValid(c.Target) {
		errorsFound = true
	}
	if c.ConfigMap == "" {
		errorsFound = true
		glog.Errorf("--configmap parameter cannot be empty")
	}
	if c.Namespace == "" {
		errorsFound = true
		glog.Errorf("--namespace parameter not set and failed to fallback")
	}
	if c.PollPeriodSeconds < 1 {
		errorsFound = true
		glog.Errorf("--poll-period-seconds cannot be less than 1")
	}

	// Log all sanity check errors before returning a single error string
	if errorsFound {
		return fmt.Errorf("failed to validate all input parameters")
	}
	return nil
}

func isTargetFormatValid(target string) bool {
	if target == "" {
		glog.Error("--target parameter cannot be empty")
		return false
	}

	splits := strings.Split(target, "/")
	resourceSplits := strings.Split(splits[0], ".")

	if len(splits) != 2 {
		glog.Error("--target must include resource and name")
		return false
	}

	if (len(resourceSplits) == 2 || len(resourceSplits) == 3) ||
		strings.HasPrefix(splits[0], "deployment") ||
		strings.HasPrefix(splits[0], "replicaset") ||
		strings.HasPrefix(splits[0], "statefulset") ||
		strings.HasPrefix(splits[0], "replicationcontroller") {
		return true
	}

	glog.Errorf("--target must include valid resource %q", resourceSplits)
	return false
}

type configMapData map[string]string

func (c *configMapData) Set(raw string) error {
	var rawData map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &rawData); err != nil {
		return err
	}
	*c = make(map[string]string)
	for key, param := range rawData {
		marshaled, err := json.Marshal(param)
		if err != nil {
			return err
		}
		(*c)[key] = string(marshaled)
	}
	return nil
}

func (c *configMapData) String() string {
	return fmt.Sprintf("%v", *c)
}

func (c *configMapData) Type() string {
	return "configMapData"
}

// AddFlags adds flags for a specific AutoScaler to the specified FlagSet
func (c *AutoScalerConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Target, "target", c.Target, "Target to scale. In format: deployment/*, replicaset/*, statefulset/* or resource.group (not case sensitive).")
	fs.StringVar(&c.ConfigMap, "configmap", c.ConfigMap, "ConfigMap containing our scaling parameters.")
	fs.StringVar(&c.Namespace, "namespace", c.Namespace, "Namespace for all operations, fallback to the namespace of this autoscaler(through MY_POD_NAMESPACE env) if not specified.")
	fs.IntVar(&c.PollPeriodSeconds, "poll-period-seconds", c.PollPeriodSeconds, "The time, in seconds, to check cluster status and perform autoscale.")
	fs.BoolVar(&c.PrintVer, "version", c.PrintVer, "Print the version and exit.")
	fs.Var(&c.DefaultParams, "default-params", "Default parameters(JSON format) for auto-scaling. Will create/re-create a ConfigMap with this default params if ConfigMap is not present.")
	fs.StringVar(&c.NodeLabels, "nodelabels", c.NodeLabels, "NodeLabels for filtering search of nodes and its cpus by LabelSelectors. Input format is a comma separated list of keyN=valueN LabelSelectors. Usage example: --nodelabels=label1=value1,label2=value2.")
	fs.IntVar(&c.MaxSyncFailures, "max-sync-failures", c.MaxSyncFailures, "Number of consecutive polling failures before exiting. Default value of 0 will allow for unlimited retries.")
}
