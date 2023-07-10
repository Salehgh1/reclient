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

// Package cppdependencyscanner encapsulates a concrete include scanner.
// It can either encapsulate clang-scan-deps or goma's input processor
// depending on build configuration.
// If specified as an argument it will alternatively connect to a remote scanner service.
package cppdependencyscanner

import (
	"context"
	"errors"

	"team/foundry-x/re-client/internal/pkg/cppdependencyscanner/depsscannerclient"
	"team/foundry-x/re-client/internal/pkg/cppdependencyscanner/includescanner"
	"team/foundry-x/re-client/internal/pkg/logger"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/command"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/filemetadata"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/outerr"

	log "github.com/golang/glog"
)

// DepsScanner is an include scanner for c++ compiles.
type DepsScanner interface {
	//ProcessInputs receives a compile command, source file, and working directory and returns
	//a list of inputs, a boolean indicating whether deps cache was used, and an error if
	//exists.
	ProcessInputs(ctx context.Context, execID string, compileCommand []string, filename, directory string, cmdEnv []string) ([]string, bool, error)
	Close()
	ShouldIgnorePlugin(plugin string) bool
}

// Executor can run commands and retrieve their outputs.
type executor interface {
	ExecuteInBackground(ctx context.Context, cmd *command.Command, oe outerr.OutErr, ch chan *command.Result) error
}

// ScannerType is the type of C++ include scanner.
type ScannerType int

const (
	// ClangScanDeps is used for include scanning.
	ClangScanDeps ScannerType = iota
	// Goma input processor is used for include scanning.
	Goma
	// GomaService indicates goma dependency scanner based input processor
	// is used for include scanning.
	GomaService
)

var (
	// ErrDepsScanTimeout is the error returned by the input processor
	// when it times out during the dependency scanning phase.
	ErrDepsScanTimeout = errors.New("cpp dependency scanner timed out")
	// UseGomaDepsScannerService indicates whether we should use
	// the goma based dependency scanner service or not.
	UseGomaDepsScannerService = false
)

// Type returns the type of include scaner being used.
func Type() ScannerType {
	if UseGomaDepsScannerService {
		return GomaService
	}
	if includescanner.Name == "Goma" {
		return Goma
	}
	return ClangScanDeps
}

// Name returns the name of include scanner used in the current binary.
func Name() string {
	if UseGomaDepsScannerService {
		return "GomaService"
	}
	return includescanner.Name
}

// New creates new DepsScanner.
func New(ctx context.Context, executor executor, fmc filemetadata.Cache, cacheDir, logDir string, cacheSizeMaxMb int, ignoredPlugins []string, useDepsCache bool, l *logger.Logger, depsScannerAddress, proxyServerAddress string) DepsScanner {
	if depsScannerAddress == "" {
		return includescanner.New(fmc, cacheDir, logDir, cacheSizeMaxMb, ignoredPlugins, useDepsCache, l)
	}
	client := depsscannerclient.New(ctx, executor, cacheDir, cacheSizeMaxMb, ignoredPlugins, useDepsCache, logDir, depsScannerAddress, proxyServerAddress)
	if client == nil {
		log.Fatal("Unable to connect to dependency scanner service")
	}
	return client
}
