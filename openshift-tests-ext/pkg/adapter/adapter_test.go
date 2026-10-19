package adapter

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"testing/fstest"
	"time"

	ext "github.com/openshift-eng/openshift-tests-extension/pkg/extension/extensiontests"
)

func TestClassifyWaitErrExitCodes(t *testing.T) {
	tests := map[string]struct {
		exitCode int
		want     ext.Result
	}{
		"pass maps to ResultPassed": {
			exitCode: 0,
			want:     ext.ResultPassed,
		},
		"fail maps to ResultFailed": {
			exitCode: 1,
			want:     ext.ResultFailed,
		},
		"skip maps to ResultSkipped": {
			exitCode: 2,
			want:     ext.ResultSkipped,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := runScenario(context.Background(), ScenarioSpec{
				Name:       name,
				Entrypoint: "/bin/bash",
				Args:       []string{"-c", "exit " + strconv.Itoa(tc.exitCode)},
			}, "", 30*time.Second)
			if result.Result != tc.want {
				t.Fatalf("got result %q, want %q (error=%q output=%q)", result.Result, tc.want, result.Error, result.Output)
			}
		})
	}
}

func TestRunScenarioSetsInteropScriptsDir(t *testing.T) {
	scriptFS := fstest.MapFS{
		"probe.sh": &fstest.MapFile{
			Data: []byte("#!/bin/bash\necho \"DIR=${INTEROP_SCRIPTS_DIR}\"\nexit 0\n"),
			Mode: 0o755,
		},
	}

	specs, err := BuildExtensionTestSpecsFromScenarios([]byte(`[
		{
			"name": "[sig-acm] scripts dir is set",
			"entrypoint": "/bin/bash",
			"args": ["-c", "${INTEROP_SCRIPTS_DIR}/probe.sh"]
		}
	]`), scriptFS)
	if err != nil {
		t.Fatalf("BuildExtensionTestSpecsFromScenarios: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("got %d specs, want 1", len(specs))
	}

	result := specs[0].Run(context.Background())
	if result.Result != ext.ResultPassed {
		t.Fatalf("got result %q error=%q output=%q", result.Result, result.Error, result.Output)
	}
	if result.Output == "" {
		t.Fatal("expected probe.sh to print INTEROP_SCRIPTS_DIR")
	}
}

func TestExtractScriptsWritesExecutableFiles(t *testing.T) {
	scriptFS := fstest.MapFS{
		"check.sh": &fstest.MapFile{Data: []byte("#!/bin/bash\nexit 0\n"), Mode: 0o644},
	}
	dir, err := extractScripts(scriptFS)
	if err != nil {
		t.Fatalf("extractScripts: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	info, err := os.Stat(filepath.Join(dir, "check.sh"))
	if err != nil {
		t.Fatalf("stat extracted script: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("extracted script is not executable: %v", info.Mode())
	}
}

func TestExtractScriptsNilFS(t *testing.T) {
	dir, err := extractScripts(nil)
	if err != nil {
		t.Fatalf("extractScripts(nil): %v", err)
	}
	if dir != "" {
		t.Fatalf("got %q, want empty dir for nil FS", dir)
	}
}
