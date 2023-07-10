// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package toolchain is responsible for determining the toolchain inputs for a specific command.
package toolchain

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"team/foundry-x/re-client/internal/pkg/inputprocessor/pathtranslator"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/command"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/filemetadata"
)

var (
	// defaultPath is the default content of the PATH variable on the bot.
	// TODO(b/149753814): make this configurable on a per action basis.
	defaultPath = []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin"}

	// metalavaRe is a regular expression to find the version number of metalava.
	metalavaRe = regexp.MustCompile(`^[\w\s]+:\s*(.+)`)
)

// InputProcessor determines the toolchain inputs of a command.
type InputProcessor struct{}

// ProcessToolchainInputs returns the toolchain inputs of the given executable path,
// and toolchains inputs.
func (p *InputProcessor) ProcessToolchainInputs(ctx context.Context, execRoot, workingDir, execPath string, toolchains []string, fmc filemetadata.Cache) (*command.InputSpec, error) {
	execPath = pathtranslator.RelToExecRoot(execRoot, workingDir, execPath)
	inp, err := p.processToolchainInputFiles(execRoot, append(toolchains, execPath), fmc)
	if err != nil {
		return nil, err
	}
	dirs := make([]string, 0)
	for _, tc := range toolchains {
		tcDir := filepath.Dir(tc)
		tcDir = pathtranslator.RelToWorkingDir(execRoot, workingDir, tcDir)
		if tcDir == "" {
			return nil, fmt.Errorf("failed to make toolchain directory [%s] relative to the working directory [%s]: %w", tcDir, workingDir, err)
		}
		dirs = append(dirs, tcDir)
	}
	if len(dirs) > 0 {
		dirs = append(dirs, defaultPath...)
		inp.EnvironmentVariables = map[string]string{"PATH": strings.Join(dirs, ":")}
	}
	return inp, nil
}
