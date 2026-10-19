// Package adapter is the local exec runner for the ACM Policy OTE binary
// in this repository. It is not a shared library.
package adapter

// ScenarioSpec is one ACM Policy OTE scenario loaded from scenarios.json.
type ScenarioSpec struct {
	// Name is the full OTE test name, e.g. "[sig-acm] ACM Policy CRD is installed".
	Name string `json:"name"`

	// Entrypoint is the program to exec (typically "/bin/bash").
	Entrypoint string `json:"entrypoint,omitempty"`

	// Args are extra argv passed to Entrypoint.
	// Example: ["-c", "${INTEROP_SCRIPTS_DIR}/check-acm-policy-crd.sh"]
	// INTEROP_SCRIPTS_DIR is set by the adapter from the embedded script FS.
	Args []string `json:"args,omitempty"`

	// WorkDir is an optional chdir before exec.
	WorkDir string `json:"workDir,omitempty"`

	// Tags become ExtensionTestSpec.Labels.
	// Use them for suite membership ("acm") and filters ("SMOKE").
	Tags []string `json:"tags,omitempty"`

	// EnvVars is the scenario parameter surface.
	// Values override ambient env.
	EnvVars map[string]string `json:"envVars,omitempty"`

	// EnvFrom copies an ambient env key onto another name before EnvVars apply.
	EnvFrom map[string]string `json:"envFrom,omitempty"`

	// IncludeCEL / ExcludeCEL restrict when the spec is selected.
	IncludeCEL string `json:"includeCEL,omitempty"`
	ExcludeCEL string `json:"excludeCEL,omitempty"`

	// Informing marks lifecycle=informing so failures do not fail the job.
	Informing bool `json:"informing,omitempty"`

	// Timeout is the adapter-enforced deadline (time.ParseDuration, e.g. "5m").
	Timeout string `json:"timeout,omitempty"`
}
