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

package reference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	_ "github.com/hyperledger/firefly/internal/coremsgs" // required to pull in translations of struct fields
	"github.com/hyperledger/firefly/pkg/core"
)

type TypeReferenceDoc struct {
	Example           []byte
	Description       []byte
	FieldDescriptions []byte
	SubFieldTables    []byte
}

/*
 * This function generates a series of markdown pages to document FireFly types, and are
 * designed to be included in the docs. Each page is a []byte value in the map, and the
 * key is the file name of the page. To add additional pages, simply create an example
 * instance of the type you would like to document, then include that in the `types`
 * array which is passed to generateMarkdownPages(). Note: It is the responsibility of
 * some other caller function to actually write the bytes to disk.
 */
func GenerateObjectsReferenceMarkdown(ctx context.Context) (map[string][]byte, error) {
	message := &core.Message{
		Header: core.MessageHeader{
			ID:        fftypes.MustParseUUID("4ea27cce-a103-4187-b318-f7b20fd87bf3"),
			Type:      core.MessageTypeBroadcast,
			Namespace: "default",
		},
		Data: []*core.DataRef{
			{
				ID: fftypes.MustParseUUID("fdf9f118-eb81-4086-a63d-b06715b3bb4e"),
			},
		},
		State: core.MessageStateConfirmed,
	}

	tokenTransfer := &core.TokenTransfer{
		Message: fftypes.MustParseUUID("855af8e7-2b02-4e05-ad7d-9ae0d4c409ba"),
		Pool:    fftypes.MustParseUUID("1244ecbe-5862-41c3-99ec-4666a18b9dd5"),
		From:    "0x98151D8AB3af082A5DC07746C220Fb6C95Bc4a50",
		To:      "0x7b746b92869De61649d148823808653430682C0d",
		Type:    core.TokenTransferTypeTransfer,
	}

	dataRef := &core.DataRef{
		ID:   fftypes.MustParseUUID("5bea782a-6cf2-4e01-95ee-cb5fa05873e9"),
		Hash: fftypes.HashString("blah"),
	}

	ffi := &core.FFI{}

	types := []interface{}{
		message,
		tokenTransfer,
		dataRef,
		ffi,
	}

	time := fftypes.FFTime{}
	bigInt := fftypes.FFBigInt{}

	simpleTypes := []interface{}{time, bigInt}

	return generateMarkdownPages(ctx, types, simpleTypes, filepath.Join("..", "..", "docs", "reference", "types"))
}

func getType(v interface{}) reflect.Type {
	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		return reflect.TypeOf(v).Elem()
	}
	return reflect.TypeOf(v)
}

func generateMarkdownPages(ctx context.Context, types []interface{}, simpleTypes []interface{}, outputPath string) (map[string][]byte, error) {
	markdownMap := make(map[string][]byte, len(types))
	rootPageNames := make([]string, len(types))
	for i, v := range types {
		rootPageNames[i] = strings.ToLower(getType(v).Name())
	}

	simpleTypesMarkdown, simpleTypesNames := generateSimpleTypesMarkdown(ctx, simpleTypes, outputPath)
	markdownMap["simpletypes"] = simpleTypesMarkdown

	for i, o := range types {
		pageTitle := getType(types[i]).Name()
		// Page index starts at 1. Simple types will be the first page. Everything else comes after that.
		pageHeader := generatePageHeader(pageTitle, i+2)
		b := bytes.NewBuffer([]byte(pageHeader))
		markdown, _, err := generateObjectReferenceMarkdown(ctx, o, reflect.TypeOf(o), rootPageNames, simpleTypesNames, []string{}, outputPath)
		if err != nil {
			return nil, err
		}
		b.Write(markdown)
		markdownMap[rootPageNames[i]] = b.Bytes()
	}
	return markdownMap, nil
}

func generateSimpleTypesMarkdown(ctx context.Context, simpleTypes []interface{}, outputPath string) ([]byte, []string) {
	simpleTypeNames := make([]string, len(simpleTypes))
	for i, v := range simpleTypes {
		simpleTypeNames[i] = strings.ToLower(getType(v).Name())
	}

	pageHeader := generatePageHeader("Simple Types", 1)

	b := bytes.NewBuffer([]byte(pageHeader))
	for _, simpleType := range simpleTypes {
		markdown, _, _ := generateObjectReferenceMarkdown(ctx, nil, reflect.TypeOf(simpleType), []string{}, simpleTypeNames, []string{}, outputPath)
		b.Write(markdown)
	}
	return b.Bytes(), simpleTypeNames
}

func generateObjectReferenceMarkdown(ctx context.Context, example interface{}, t reflect.Type, rootPageNames, simpleTypeNames, generatedTableNames []string, outputPath string) ([]byte, []string, error) {
	typeReferenceDoc := TypeReferenceDoc{}

	if t.Kind() == reflect.Ptr {
		t = reflect.TypeOf(example).Elem()
	}
	// generatedTableNames is where we keep track of all the tables we've generated (recursively)
	// for creating hyperlinks within the markdown
	generatedTableNames = append(generatedTableNames, strings.ToLower(t.Name()))

	// If a detailed type_description.md file exists, include that in a Description section here
	if _, err := os.Stat(filepath.Join(outputPath, "includes", fmt.Sprintf("%s_description.md", strings.ToLower(t.Name())))); err == nil {
		typeReferenceDoc.Description = []byte(fmt.Sprintf("{%% include_relative includes/%s_description.md %%}\n\n", strings.ToLower(t.Name())))
	}

	// Include an example JSON representation if we have one available
	if example != nil {
		exampleJSON, err := json.MarshalIndent(example, "", "    ")
		if err != nil {
			return nil, nil, err
		}
		typeReferenceDoc.Example = exampleJSON
	}

	// If the type is a struct, look into each field inside it
	if t.Kind() == reflect.Struct {
		typeReferenceDoc.FieldDescriptions, typeReferenceDoc.SubFieldTables, generatedTableNames = generateFieldDescriptionsForStruct(ctx, t, rootPageNames, simpleTypeNames, generatedTableNames, outputPath)
	}

	// buff is the main buffer where we will write the markdown for this page
	buff := bytes.NewBuffer([]byte{})
	buff.WriteString(fmt.Sprintf("## %s\n\n", t.Name()))

	// If we only have one section, we will not write H3 headers
	sectionCount := 0
	if typeReferenceDoc.Description != nil {
		sectionCount++
	}
	if typeReferenceDoc.Example != nil {
		sectionCount++
	}
	if typeReferenceDoc.FieldDescriptions != nil {
		sectionCount++
	}

	if typeReferenceDoc.Description != nil {
		buff.Write(typeReferenceDoc.Description)
	}
	if typeReferenceDoc.Example != nil && len(typeReferenceDoc.Example) > 0 {
		if sectionCount > 1 {
			buff.WriteString("### Example\n\n```json\n")
		}
		buff.Write(typeReferenceDoc.Example)
		buff.WriteString("\n```\n\n")
	}
	if typeReferenceDoc.FieldDescriptions != nil && len(typeReferenceDoc.FieldDescriptions) > 0 {
		if sectionCount > 1 {
			buff.WriteString("### Field Descriptions\n\n")
		}
		buff.Write(typeReferenceDoc.FieldDescriptions)
		buff.WriteString("\n")
	}

	if typeReferenceDoc.SubFieldTables != nil && len(typeReferenceDoc.SubFieldTables) > 0 {
		buff.Write(typeReferenceDoc.SubFieldTables)
	}

	return buff.Bytes(), generatedTableNames, nil
}

func generateFieldDescriptionsForStruct(ctx context.Context, t reflect.Type, rootPageNames, simpleTypeNames, generatedTableNames []string, outputPath string) ([]byte, []byte, []string) {
	fieldDescriptionsBytes := []byte{}
	// subFieldBuff is where we write any additional tables for sub fields that may be on this struct
	subFieldBuff := bytes.NewBuffer([]byte{})
	numField := t.NumField()
	if numField > 0 {
		// Write the table to a temporary buffer - we will throw it away if there are no
		// public JSON serializable fields on the struct
		tableRowCount := 0
		tableBuff := bytes.NewBuffer([]byte{})
		tableBuff.WriteString("| Field Name | Description | Type |\n")
		tableBuff.WriteString("|------------|-------------|------|\n")
		for i := 0; i < numField; i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			ffstructTag := field.Tag.Get("ffstruct")
			ffexcludeTag := field.Tag.Get("ffexclude")

			// If the field is specifically excluded, or doesn't have a json tag, skip it
			if ffexcludeTag != "" || jsonTag == "" || jsonTag == "-" {
				continue
			}

			jsonFieldName := strings.Split(jsonTag, ",")[0]
			messageKeyName := fmt.Sprintf("%s.%s", ffstructTag, jsonFieldName)
			description := i18n.Expand(ctx, i18n.MessageKey(messageKeyName))
			isArray := false

			fieldType := field.Type
			fireflyType := fieldType.Name()

			if fieldType.Kind() == reflect.Slice {
				fieldType = fieldType.Elem()
				fireflyType = fieldType.Name()
				isArray = true
			}

			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
				fireflyType = fieldType.Name()
			}

			if isArray {
				fireflyType = fmt.Sprintf("%s[]", fireflyType)
			}

			fireflyType = fmt.Sprintf("`%s`", fireflyType)

			if fieldType.Kind() == reflect.Struct {
				fieldInRootPages := false
				fieldInSimpleTypes := false
				for _, rootPageName := range rootPageNames {
					if strings.ToLower(fieldType.Name()) == rootPageName {
						fieldInRootPages = true
						break
					}
				}
				for _, simpleTypeName := range simpleTypeNames {
					if strings.ToLower(fieldType.Name()) == simpleTypeName {
						fieldInSimpleTypes = true
						break
					}
				}

				link := fmt.Sprintf("#%s", strings.ToLower(fieldType.Name()))
				if fieldInRootPages {
					link = fmt.Sprintf("%s#%s", fieldType.Name(), strings.ToLower(fieldType.Name()))
				} else if fieldInSimpleTypes {
					link = fmt.Sprintf("simpletypes#%s", strings.ToLower(fieldType.Name()))
				}
				fireflyType = fmt.Sprintf("[%s](%s)", fireflyType, link)

				// Generate the table for the sub type
				tableAlreadyGenerated := false
				for _, tableName := range generatedTableNames {
					if strings.ToLower(fieldType.Name()) == tableName {
						tableAlreadyGenerated = true
						break
					}
				}
				if !tableAlreadyGenerated && !fieldInRootPages && !fieldInSimpleTypes {
					subFieldMarkdown, newTableNames, _ := generateObjectReferenceMarkdown(ctx, nil, fieldType, rootPageNames, simpleTypeNames, generatedTableNames, outputPath)
					generatedTableNames = newTableNames
					subFieldBuff.Write(subFieldMarkdown)
					subFieldBuff.WriteString("\n")
				}
			}
			tableBuff.WriteString(fmt.Sprintf("| %s | %s | %s |\n", jsonFieldName, description, fireflyType))
			tableRowCount++
		}
		if tableRowCount > 1 {
			fieldDescriptionsBytes = tableBuff.Bytes()
		}
	}
	return fieldDescriptionsBytes, subFieldBuff.Bytes(), generatedTableNames
}

func generatePageHeader(pageTitle string, navOrder int) string {
	return fmt.Sprintf(`---
layout: default
title: %s
parent: Types
grand_parent: Reference
nav_order: %v
---

# %s
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
`, pageTitle, navOrder, pageTitle)
}
