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

package i18n

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

// MessageKey is the lookup string for any translation
type MessageKey string

// ErrorMessageKey is a special lookup string conforming to FireFly's rules for error message registration
type ErrorMessageKey MessageKey

// Expand for use in docs and logging - returns a translated message, translated the language of the context
func Expand(ctx context.Context, key MessageKey, inserts ...interface{}) string {
	return pFor(ctx).Sprintf(string(key), inserts...)
}

// ExpandWithCode for use in error scenarios - returns a translated message with a "MSG012345:" prefix, translated the language of the context
func ExpandWithCode(ctx context.Context, key MessageKey, inserts ...interface{}) string {
	return string(key) + ": " + pFor(ctx).Sprintf(string(key), inserts...)
}

// WithLang sets the language on the context
func WithLang(ctx context.Context, lang language.Tag) context.Context {
	return context.WithValue(ctx, ctxLangKey{}, lang)
}

type (
	ctxLangKey struct{}
)

// en language is special, as the definition of each key comes directly with the english language description
var enLang = language.MustParse("en")

var serverLangs = []language.Tag{
	language.AmericanEnglish, // Only English currently supported
}

var langMatcher = language.NewMatcher(serverLangs)

var statusHints = map[string]int{}
var msgIDUniq = map[string]bool{}

// FFE is the translations helper to register an error message
func FFE(key, enTranslation string, statusHint ...int) ErrorMessageKey {
	validPrefix := false
	for rp := range registeredPrefixes {
		if strings.HasPrefix(key, rp) {
			validPrefix = true
			break
		}
	}
	if !validPrefix {
		panic(fmt.Sprintf("Message ID %s does not use one of the registered prefixes in the common utility package", key))
	}
	msgID := FFM(key, enTranslation)
	if len(statusHint) > 0 {
		statusHints[key] = statusHint[0]
	}
	return ErrorMessageKey(msgID)
}

// FFM is the enTranslations helper to define a new message (not used in translation files)
func FFM(key, enTranslation string) MessageKey {
	if _, exists := msgIDUniq[key]; exists {
		panic(fmt.Sprintf("Message ID %s re-used", key))
	}
	msgIDUniq[key] = true
	_ = message.Set(enLang, key, catalog.String(enTranslation))
	return MessageKey(key)
}

var defaultLangPrinter *message.Printer

func pFor(ctx context.Context) *message.Printer {
	lang := ctx.Value(ctxLangKey{})
	if lang == nil {
		return defaultLangPrinter
	}
	return message.NewPrinter(lang.(language.Tag))
}

func init() {
	SetLang("en")
	msgIDUniq = map[string]bool{} // Clear out that memory as no longer needed
}

func SetLang(lang string) {
	// Allow a lang var to be used
	tag, _, _ := langMatcher.Match(language.Make(lang))
	defaultLangPrinter = message.NewPrinter(tag)
}

func GetStatusHint(code string) (int, bool) {
	i, ok := statusHints[code]
	return i, ok
}
