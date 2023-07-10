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

package auth

import (
	"os"
	"time"

	apb "team/foundry-x/re-client/api/auth"

	log "github.com/golang/glog"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/encoding/prototext"
	tspb "google.golang.org/protobuf/types/known/timestamppb"
)

// CachedCredentials are the credentials cached to disk.
type cachedCredentials struct {
	m     Mechanism
	token *oauth2.Token
}

func loadFromDisk(tf string) (cachedCredentials, error) {
	if tf == "" {
		return cachedCredentials{}, nil
	}
	blob, err := os.ReadFile(tf)
	if err != nil {
		return cachedCredentials{}, err
	}
	cPb := &apb.Credentials{}
	if err := prototext.Unmarshal(blob, cPb); err != nil {
		return cachedCredentials{}, err
	}
	accessToken := cPb.GetToken()
	exp := TimeFromProto(cPb.GetExpiry())
	var token *oauth2.Token
	if accessToken != "" && !exp.IsZero() {
		token = &oauth2.Token{
			AccessToken: accessToken,
			Expiry:      exp,
		}
	}
	c := cachedCredentials{
		m:     protoToMechanism(cPb.GetMechanism()),
		token: token,
	}
	log.Infof("Loaded cached credentials of type %v, expires at %v", c.m, exp)
	return c, nil
}

func saveToDisk(c cachedCredentials, tf string) error {
	if tf == "" {
		return nil
	}
	cPb := &apb.Credentials{}
	cPb.Mechanism = mechanismToProto(c.m)
	if c.token != nil {
		cPb.Token = c.token.AccessToken
		cPb.Expiry = TimeToProto(c.token.Expiry)
	}
	f, err := os.Create(tf)
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString(prototext.Format(cPb))
	log.Infof("Saved cached credentials of type %v, expires at %v to %v", c.m, cPb.Expiry, tf)
	return nil
}

func mechanismToProto(m Mechanism) apb.AuthMechanism_Value {
	switch m {
	case Unknown:
		return apb.AuthMechanism_UNSPECIFIED
	case ADC:
		return apb.AuthMechanism_ADC
	case GCE:
		return apb.AuthMechanism_GCE
	case CredentialFile:
		return apb.AuthMechanism_CREDENTIAL_FILE
	case None:
		return apb.AuthMechanism_NONE
	default:
		return apb.AuthMechanism_UNSPECIFIED
	}
}

func protoToMechanism(p apb.AuthMechanism_Value) Mechanism {
	switch p {
	case apb.AuthMechanism_UNSPECIFIED:
		return Unknown
	case apb.AuthMechanism_ADC:
		return ADC
	case apb.AuthMechanism_GCE:
		return GCE
	case apb.AuthMechanism_NONE:
		return None
	case apb.AuthMechanism_CREDENTIAL_FILE:
		return CredentialFile
	default:
		return Unknown
	}
}

// TimeToProto converts a valid time.Time into a proto Timestamp.
func TimeToProto(t time.Time) *tspb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return tspb.New(t)
}

// TimeFromProto converts a valid Timestamp proto into a time.Time.
func TimeFromProto(tPb *tspb.Timestamp) time.Time {
	if tPb == nil {
		return time.Time{}
	}
	return tPb.AsTime()
}
