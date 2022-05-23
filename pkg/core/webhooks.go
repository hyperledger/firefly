// Copyright © 2022 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
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

package core

type WebhookSubOptions struct {
	Fastack  bool                `ffstruct:"WebhookSubOptions" json:"fastack,omitempty"`
	URL      string              `ffstruct:"WebhookSubOptions" json:"url,omitempty"`
	Method   string              `ffstruct:"WebhookSubOptions" json:"method,omitempty"`
	JSON     bool                `ffstruct:"WebhookSubOptions" json:"json,omitempty"`
	Reply    bool                `ffstruct:"WebhookSubOptions" json:"reply,omitempty"`
	ReplyTag string              `ffstruct:"WebhookSubOptions" json:"replytag,omitempty"`
	ReplyTX  string              `ffstruct:"WebhookSubOptions" json:"replytx,omitempty"`
	Headers  map[string]string   `ffstruct:"WebhookSubOptions" json:"headers,omitempty"`
	Query    map[string]string   `ffstruct:"WebhookSubOptions" json:"query,omitempty"`
	Input    WebhookInputOptions `ffstruct:"WebhookSubOptions" json:"input,omitempty"`
}

type WebhookInputOptions struct {
	Query   string `ffstruct:"WebhookInputOptions" json:"query,omitempty"`
	Headers string `ffstruct:"WebhookInputOptions" json:"headers,omitempty"`
	Body    string `ffstruct:"WebhookInputOptions" json:"body,omitempty"`
	Path    string `ffstruct:"WebhookInputOptions" json:"path,omitempty"`
	ReplyTX string `ffstruct:"WebhookInputOptions" json:"replytx,omitempty"`
}
