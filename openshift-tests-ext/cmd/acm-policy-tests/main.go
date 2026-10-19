package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/openshift-eng/openshift-tests-extension/pkg/cmd"
	e "github.com/openshift-eng/openshift-tests-extension/pkg/extension"
	"github.com/stolostron/policy-collection/openshift-tests-ext/pkg/adapter"
	"github.com/stolostron/policy-collection/openshift-tests-ext/scripts"
	"github.com/stolostron/policy-collection/openshift-tests-ext/test/acm"
)

func main() {
	defaultTimeout := 5 * time.Minute

	registry := e.NewRegistry()
	ext := e.NewExtension("acm", "external", "acm-policy-tests")

	ext.AddSuite(e.Suite{
		Name:             "acm/all",
		Description:      "All ACM Policy tests",
		Qualifiers:       []string{`tags["component"] == "acm"`},
		ClusterStability: e.ClusterStabilityStable,
		Parallelism:      1,
		TestTimeout:      &defaultTimeout,
	})

	ext.AddSuite(e.Suite{
		Name:             "acm/smoke",
		Description:      "ACM Policy smoke tests",
		Qualifiers:       []string{`labels.exists(l, l=="SMOKE")`},
		ClusterStability: e.ClusterStabilityStable,
		Parallelism:      1,
		TestTimeout:      &defaultTimeout,
	})

	specs, err := adapter.BuildExtensionTestSpecsFromScenarios(acm.ACMPolicyScenarios, scripts.FS)
	if err != nil {
		panic(fmt.Sprintf("couldn't build extension test specs from ACM Policy scenarios: %+v", err.Error()))
	}

	ext.AddSpecs(specs)
	registry.Register(ext)

	root := &cobra.Command{
		Use:   "acm-policy-tests",
		Short: "ACM Policy Tests Extension for OpenShift",
		Long:  "OpenShift Tests Extension (OTE) binary for ACM Policy testing",
	}

	root.AddCommand(cmd.DefaultExtensionCommands(registry)...)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
