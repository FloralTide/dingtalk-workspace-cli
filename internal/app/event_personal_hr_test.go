// Copyright 2026 Alibaba Group
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

package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/event/personal"
)

func TestCrossPlatformCoveragePersonalHRMEventListSchemaAndValidation(t *testing.T) {
	eventKeys := []string{
		personal.EventHRMRegularLifecycleChanged,
		personal.EventHRMTransferLifecycleChanged,
	}

	list := newEventListCommand()
	list.SilenceUsage = true
	list.SilenceErrors = true
	var listOut bytes.Buffer
	list.SetOut(&listOut)
	list.SetArgs([]string{"--category", "hr"})
	if err := list.Execute(); err != nil {
		t.Fatalf("event list --category hr error = %v", err)
	}
	for _, eventKey := range eventKeys {
		if !strings.Contains(listOut.String(), eventKey) {
			t.Fatalf("HR event list missing %s:\n%s", eventKey, listOut.String())
		}
	}
	if strings.Contains(listOut.String(), personal.EventMention) || strings.Contains(listOut.String(), personal.EventOAApprovalTaskCreated) {
		t.Fatalf("HR category list leaked another category:\n%s", listOut.String())
	}

	for _, eventKey := range eventKeys {
		t.Run(eventKey+"/schema", func(t *testing.T) {
			schema := newEventSchemaCommand()
			schema.SilenceUsage = true
			schema.SilenceErrors = true
			var schemaOut bytes.Buffer
			schema.SetOut(&schemaOut)
			schema.SetArgs([]string{eventKey, "--flatten"})
			if err := schema.Execute(); err != nil {
				t.Fatalf("event schema %s --flatten error = %v", eventKey, err)
			}
			var doc map[string]any
			if err := json.Unmarshal(schemaOut.Bytes(), &doc); err != nil {
				t.Fatalf("decode HR schema: %v\n%s", err, schemaOut.String())
			}
			if doc["event_key"] != eventKey || doc["category"] != "hr" || doc["rule_type"] != "all" || doc["jq_root_path"] != "." {
				t.Fatalf("HR schema document = %#v", doc)
			}
			schemaBody, ok := doc["schema"].(map[string]any)
			if !ok {
				t.Fatalf("HR schema body = %#v", doc["schema"])
			}
			properties, ok := schemaBody["properties"].(map[string]any)
			if !ok || len(properties) != 5 {
				t.Fatalf("HR schema properties = %#v", schemaBody["properties"])
			}
			for _, name := range []string{"type", "event_id", "timestamp", "subscribe_id", "payload"} {
				if _, ok := properties[name].(map[string]any); !ok {
					t.Fatalf("HR schema property %s = %#v", name, properties[name])
				}
			}
		})

		if err := validatePersonalBusinessEventOptions(eventKey, personalConsumeOptions{}); err != nil {
			t.Fatalf("HR event without target/filter options error = %v", err)
		}
		prepared, err := preparePersonalSubscription(personal.Identity{}, personalConsumeOptions{EventKey: eventKey})
		if err != nil {
			t.Fatalf("prepare HR subscription error = %v", err)
		}
		if prepared.RuleType != "all" || prepared.Request.RuleParam != nil || prepared.Request.Filter != nil {
			t.Fatalf("prepared HR subscription = %#v, want all rule without filterRule", prepared)
		}
		for name, opts := range map[string]personalConsumeOptions{
			"--user":             {UserID: "user-1"},
			"--open-dingtalk-id": {OpenDingTalkID: "open-user-1"},
			"--group":            {GroupID: "cid-1"},
			"--query":            {QueryCSV: "urgent"},
			"--filter-json":      {FilterJSON: `{"field":"content","op":"eq","value":"urgent"}`},
		} {
			err := validatePersonalBusinessEventOptions(eventKey, opts)
			if err == nil || !strings.Contains(err.Error(), name+" not supported for HR event "+eventKey) {
				t.Fatalf("HR %s validation error = %v", name, err)
			}
		}
	}
}
