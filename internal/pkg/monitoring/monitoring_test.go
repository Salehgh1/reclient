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

package monitoring

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	lpb "team/foundry-x/re-client/api/log"
	"team/foundry-x/re-client/internal/pkg/auth"
	st "team/foundry-x/re-client/internal/pkg/stats"
	"team/foundry-x/re-client/pkg/version"

	"contrib.go.opencensus.io/exporter/stackdriver"
	cpb "github.com/bazelbuild/remote-apis-sdks/go/api/command"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/command"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

func TestExportMetrics(t *testing.T) {
	t1 := time.Now()
	t2 := t1.Add(time.Second)
	t3 := t1.Add(10 * time.Second)
	recs := []*lpb.LogRecord{
		&lpb.LogRecord{
			Result: &cpb.CommandResult{Status: cpb.CommandResultStatus_CACHE_HIT},
			RemoteMetadata: &lpb.RemoteMetadata{
				Result: &cpb.CommandResult{Status: cpb.CommandResultStatus_CACHE_HIT},
			},
			LocalMetadata: &lpb.LocalMetadata{
				EventTimes: map[string]*cpb.TimeInterval{
					"ProxyExecution": &cpb.TimeInterval{
						From: command.TimeToProto(t1),
						To:   command.TimeToProto(t2),
					},
				},
				Labels: map[string]string{"type": "tool"},
			},
		},
		&lpb.LogRecord{
			Result: &cpb.CommandResult{Status: cpb.CommandResultStatus_SUCCESS},
			RemoteMetadata: &lpb.RemoteMetadata{
				Result: &cpb.CommandResult{Status: cpb.CommandResultStatus_SUCCESS},
			},
			LocalMetadata: &lpb.LocalMetadata{
				EventTimes: map[string]*cpb.TimeInterval{
					"ProxyExecution": &cpb.TimeInterval{
						From: command.TimeToProto(t1),
						To:   command.TimeToProto(t3),
					},
				},
				Labels: map[string]string{"type": "tool"},
			},
		},
		&lpb.LogRecord{
			Result: &cpb.CommandResult{Status: cpb.CommandResultStatus_SUCCESS},
			RemoteMetadata: &lpb.RemoteMetadata{
				Result: &cpb.CommandResult{Status: cpb.CommandResultStatus_NON_ZERO_EXIT},
			},
			LocalMetadata: &lpb.LocalMetadata{
				Result: &cpb.CommandResult{Status: cpb.CommandResultStatus_SUCCESS},
				EventTimes: map[string]*cpb.TimeInterval{
					"ProxyExecution": &cpb.TimeInterval{
						From: command.TimeToProto(t1),
						To:   command.TimeToProto(t3),
					},
				},
				Labels: map[string]string{"type": "tool"},
			},
		},
	}
	s := st.NewFromRecords(recs, nil)
	sp := s.ToProto()
	r := &stubRecorder{reports: make([]*metricReport, 0)}
	e := &Exporter{
		project:         "fake-project",
		recorder:        r,
		authCredentials: &auth.Credentials{},
	}
	err := e.initCloudMonitoring(context.Background())
	if err != nil {
		t.Errorf("Failed to initialize cloud monitoring: %v", err)
	}
	e.ExportBuildMetrics(context.Background(), sp)
	for _, r := range recs {
		e.ExportActionMetrics(context.Background(), r)
	}
	e.Close()
	wantReports := []*metricReport{
		&metricReport{
			Name: ActionCount.Name(),
			Val:  1,
			Tags: map[string]string{
				labelsKey.Name():       "[type=tool]",
				osFamilyKey.Name():     runtime.GOOS,
				versionKey.Name():      version.CurrentVersion(),
				remoteStatusKey.Name(): "CACHE_HIT",

				statusKey.Name(): "CACHE_HIT",
			},
		},
		&metricReport{
			Name: ActionLatency.Name(),
			Val:  1000,
			Tags: map[string]string{
				labelsKey.Name():       "[type=tool]",
				osFamilyKey.Name():     runtime.GOOS,
				versionKey.Name():      version.CurrentVersion(),
				remoteStatusKey.Name(): "CACHE_HIT",
				statusKey.Name():       "CACHE_HIT",
			},
		},
		&metricReport{
			Name: ActionCount.Name(),
			Val:  1,
			Tags: map[string]string{
				labelsKey.Name():       "[type=tool]",
				osFamilyKey.Name():     runtime.GOOS,
				versionKey.Name():      version.CurrentVersion(),
				remoteStatusKey.Name(): "SUCCESS",
				statusKey.Name():       "SUCCESS",
			},
		},
		&metricReport{
			Name: ActionLatency.Name(),
			Val:  10000,
			Tags: map[string]string{
				labelsKey.Name():       "[type=tool]",
				osFamilyKey.Name():     runtime.GOOS,
				versionKey.Name():      version.CurrentVersion(),
				remoteStatusKey.Name(): "SUCCESS",
				statusKey.Name():       "SUCCESS",
			},
		},
		&metricReport{
			Name: ActionCount.Name(),
			Val:  1,
			Tags: map[string]string{
				labelsKey.Name():       "[type=tool]",
				osFamilyKey.Name():     runtime.GOOS,
				versionKey.Name():      version.CurrentVersion(),
				remoteStatusKey.Name(): "NON_ZERO_EXIT",
				statusKey.Name():       "SUCCESS",
			},
		},
		&metricReport{
			Name: ActionLatency.Name(),
			Val:  10000,
			Tags: map[string]string{
				labelsKey.Name():       "[type=tool]",
				osFamilyKey.Name():     runtime.GOOS,
				versionKey.Name():      version.CurrentVersion(),
				remoteStatusKey.Name(): "NON_ZERO_EXIT",
				statusKey.Name():       "SUCCESS",
			},
		},
		&metricReport{
			Name: BuildCount.Name(),
			Val:  1,
			Tags: map[string]string{
				osFamilyKey.Name(): runtime.GOOS,
				versionKey.Name():  version.CurrentVersion(),
				statusKey.Name():   "SUCCESS",
			},
		},
		&metricReport{
			Name: BuildLatency.Name(),
			Val:  10,
			Tags: map[string]string{
				osFamilyKey.Name(): runtime.GOOS,
				versionKey.Name():  version.CurrentVersion(),
			},
		},
		&metricReport{
			Name: BuildCacheHitRatio.Name(),
			Val:  1.0 / 3.0,
			Tags: map[string]string{
				osFamilyKey.Name(): runtime.GOOS,
				versionKey.Name():  version.CurrentVersion(),
			},
		},
	}
	repCmp := cmpopts.SortSlices(func(a, b *metricReport) bool {
		return a.hash() < b.hash()
	})
	if diff := cmp.Diff(wantReports, r.reports, repCmp); diff != "" {
		t.Errorf("Recorded metrics have diff: (-want +got)\n%s", diff)
	}
}

func TestExportBuildFailureMetrics(t *testing.T) {
	t1 := time.Now()
	t2 := t1.Add(time.Second)
	recs := []*lpb.LogRecord{
		&lpb.LogRecord{
			Result: &cpb.CommandResult{Status: cpb.CommandResultStatus_CACHE_HIT},
			RemoteMetadata: &lpb.RemoteMetadata{
				Result: &cpb.CommandResult{Status: cpb.CommandResultStatus_CACHE_HIT},
			},
			LocalMetadata: &lpb.LocalMetadata{
				EventTimes: map[string]*cpb.TimeInterval{
					"ProxyExecution": &cpb.TimeInterval{
						From: command.TimeToProto(t1),
						To:   command.TimeToProto(t2),
					},
				},
				Labels: map[string]string{"type": "tool"},
			},
		},
	}
	s := st.NewFromRecords(recs, nil)
	sp := s.ToProto()
	logDir := t.TempDir()
	r := &stubRecorder{reports: make([]*metricReport, 0)}
	e := &Exporter{
		project:         "fake-project",
		recorder:        r,
		logDir:          logDir,
		authCredentials: &auth.Credentials{},
	}
	err := e.initCloudMonitoring(context.Background())
	if err != nil {
		t.Errorf("Failed to initialize cloud monitoring: %v", err)
	}
	logFile := "reproxy.FATAL"
	if runtime.GOOS == "windows" {
		logFile = "reproxy.exe.FATAL"
	}
	os.WriteFile(filepath.Join(logDir, logFile), []byte("FATAL"), 0666)
	e.ExportBuildMetrics(context.Background(), sp)
	for _, r := range recs {
		e.ExportActionMetrics(context.Background(), r)
	}
	e.Close()
	wantReports := []*metricReport{
		&metricReport{
			Name: ActionCount.Name(),
			Val:  1,
			Tags: map[string]string{
				labelsKey.Name():       "[type=tool]",
				osFamilyKey.Name():     runtime.GOOS,
				versionKey.Name():      version.CurrentVersion(),
				remoteStatusKey.Name(): "CACHE_HIT",
				statusKey.Name():       "CACHE_HIT",
			},
		},
		&metricReport{
			Name: ActionLatency.Name(),
			Val:  1000,
			Tags: map[string]string{
				labelsKey.Name():       "[type=tool]",
				osFamilyKey.Name():     runtime.GOOS,
				versionKey.Name():      version.CurrentVersion(),
				remoteStatusKey.Name(): "CACHE_HIT",
				statusKey.Name():       "CACHE_HIT",
			},
		},
		&metricReport{
			Name: BuildCount.Name(),
			Val:  1,
			Tags: map[string]string{
				osFamilyKey.Name(): runtime.GOOS,
				versionKey.Name():  version.CurrentVersion(),
				statusKey.Name():   "FAILURE",
			},
		},
		&metricReport{
			Name: BuildLatency.Name(),
			Val:  1,
			Tags: map[string]string{
				osFamilyKey.Name(): runtime.GOOS,
				versionKey.Name():  version.CurrentVersion(),
			},
		},
		&metricReport{
			Name: BuildCacheHitRatio.Name(),
			Val:  1.0,
			Tags: map[string]string{
				osFamilyKey.Name(): runtime.GOOS,
				versionKey.Name():  version.CurrentVersion(),
			},
		},
	}
	repCmp := cmpopts.SortSlices(func(a, b *metricReport) bool {
		return a.hash() < b.hash()
	})
	if diff := cmp.Diff(wantReports, r.reports, repCmp); diff != "" {
		t.Errorf("Recorded metrics have diff: (-want +got)\n%s", diff)
	}
}
func TestInitCloudMonitoringError(t *testing.T) {
	r := &stubRecorder{
		reports: make([]*metricReport, 0),
		err:     errors.New("fake error"),
	}
	e := &Exporter{
		project:         "fake-project",
		recorder:        r,
		authCredentials: &auth.Credentials{},
	}
	if err := e.initCloudMonitoring(context.Background()); err == nil {
		t.Errorf("initCloudMonitoring succeeded; expected failure")
	}
	e.Close()
}

type metricReport struct {
	Name string
	Val  float64
	Tags map[string]string
}

func (m *metricReport) hash() string {
	var ks []string
	for k, v := range m.Tags {
		ks = append(ks, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(ks)
	return fmt.Sprintf("%s%v[%s]", m.Name, m.Val, strings.Join(ks, ","))
}

type stubRecorder struct {
	stackDriverRecorder
	reports []*metricReport
	err     error
}

func (s *stubRecorder) initialize(o stackdriver.Options) error {
	return s.err
}

func (s *stubRecorder) close() {}

func (s *stubRecorder) tagsContext(ctx context.Context, labels map[tag.Key]string) context.Context {
	return s.stackDriverRecorder.tagsContext(ctx, labels)
}

func (s *stubRecorder) recordWithTags(ctx context.Context, labels map[tag.Key]string, val stats.Measurement) {
	tagVals := make(map[string]string)
	for _, k := range []tag.Key{osFamilyKey, versionKey, remoteStatusKey, statusKey, labelsKey} {
		v, ok := tag.FromContext(ctx).Value(k)
		if !ok {
			v, ok = labels[k]
			if !ok {
				continue
			}
		}
		tagVals[k.Name()] = v
	}
	s.reports = append(s.reports, &metricReport{
		Name: val.Measure().Name(),
		Val:  val.Value(),
		Tags: tagVals,
	})
}
