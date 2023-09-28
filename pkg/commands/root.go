package commands

import (
	"errors"
	"fmt"

	jsonengine "github.com/eddycharly/json-kyverno/pkg/json-engine"
	"github.com/eddycharly/json-kyverno/pkg/payload"
	"github.com/eddycharly/json-kyverno/pkg/policy"
	"github.com/eddycharly/json-kyverno/pkg/template"
	"github.com/kyverno/kyverno/cmd/cli/kubectl-kyverno/output/pluralize"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type command struct {
	payload       string
	preprocessors []string
	policies      []string
}

func (c *command) Run(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Loading policies ...")
	policies, err := policy.Load(c.policies...)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "Loading plan ...")
	payload, err := payload.Load(c.payload)
	if err != nil {
		return err
	}
	if payload == nil {
		return errors.New("payload is `null`")
	}
	fmt.Fprintln(out, "Pre processing ...")
	for _, preprocessor := range c.preprocessors {
		result, err := template.Execute(preprocessor, payload)
		if err != nil {
			return err
		}
		if result == nil {
			return fmt.Errorf("prepocessor resulted in `null` payload (%s)", preprocessor)
		}
		payload = result
	}
	var resources []interface{}
	if slice, ok := payload.([]interface{}); ok {
		resources = slice
	} else {
		resources = append(resources, payload)
	}
	fmt.Fprintln(out, "Running", "(", "evaluating", len(resources), pluralize.Pluralize(len(resources), "resource", "resources"), "against", len(policies), pluralize.Pluralize(len(policies), "policy", "policies"), ")", "...")
	e := jsonengine.New()
	responses := e.Run(jsonengine.JsonEngineRequest{
		Resources: resources,
		Policies:  policies,
	})
	for _, response := range responses {
		resourceName, _, _ := unstructured.NestedString(response.Resource.(map[string]interface{}), "address")
		if response.Error == nil {
			fmt.Fprintln(out, "-", response.Policy.Name, "/", response.Rule.Name, "/", resourceName, "PASSED")
		} else {
			fmt.Fprintln(out, "-", response.Policy.Name, "/", response.Rule.Name, "/", resourceName, "FAILED:", response.Error)
		}
	}
	fmt.Fprintln(out, "Done")
	return nil
}

func NewRootCommand() *cobra.Command {
	var command command
	cmd := &cobra.Command{
		Use:          "json-kyverno",
		Short:        "json-kyverno",
		Long:         "json-kyverno is a CLI tool to apply policies to json resources",
		Args:         cobra.NoArgs,
		RunE:         command.Run,
		SilenceUsage: true,
	}
	cmd.Flags().StringVar(&command.payload, "payload", "", "Path to json payload")
	cmd.Flags().StringSliceVar(&command.preprocessors, "pre-process", nil, "JmesPath expression used to pre process payload")
	cmd.Flags().StringSliceVar(&command.policies, "policy", nil, "Path to json-kyverno policies")
	return cmd
}
