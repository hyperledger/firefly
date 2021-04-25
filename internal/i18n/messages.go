// Copyright © 2021 Kaleido, Inc.
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

package i18n

import (
	"context"
	"fmt"

	"github.com/kaleido-io/firefly/internal/config"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// MessageKey is assigned to each unique message in the system
type MessageKey int

const (
	// MsgConfigFailed - startup failed
	MsgConfigFailed MessageKey = 10101
)

// Sprintf returns a translated message, for the language of the specified context
func Sprintf(ctx context.Context, key MessageKey, inserts ...interface{}) string {
	return pFor(ctx).Sprintf(key.String(), inserts...)
}

// WithLang sets the language on the context
func WithLang(ctx context.Context, lang language.Tag) context.Context {
	return context.WithValue(ctx, ctxLangKey{}, lang)
}

type (
	ctxLangKey struct{}
)

type msg struct {
	msgid       MessageKey
	localString string
}

type lang struct {
	tag      string
	messages []msg
}

var serverLangs = []language.Tag{
	language.AmericanEnglish, // Only English currently supported
}

var langMatcher = language.NewMatcher(serverLangs)

var all = [...]lang{
	{"en", enTranslations},
}

var defaultLangPrinter *message.Printer

func (m MessageKey) String() string {
	return fmt.Sprintf("MSG%d", m)
}

func pFor(ctx context.Context) *message.Printer {
	lang := ctx.Value(ctxLangKey{})
	if lang == nil {
		return defaultLangPrinter
	}
	return message.NewPrinter(lang.(language.Tag))
}

func init() {
	for _, e := range all {
		tag := language.MustParse(e.tag)
		for _, msg := range e.messages {
			message.SetString(tag, msg.msgid.String(), fmt.Sprintf("%s: %s", msg.msgid, msg.localString))
		}
	}
	// Allow a lang var to be used
	lang := config.GetString(config.Lang)
	tag, _, _ := langMatcher.Match(language.Make(lang))
	defaultLangPrinter = message.NewPrinter(tag)
}
