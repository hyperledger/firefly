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

package sqlcommon

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/persistence"
)

func filterSelect(ctx context.Context, sel sq.SelectBuilder, filter persistence.Filter, typeMap map[string]string) (sq.SelectBuilder, error) {
	fi, err := filter.Finalize()
	if err != nil {
		return sel, err
	}
	if len(fi.Sort) == 0 {
		fi.Sort = []string{"seq"}
		fi.Descending = true
	}
	return filterSelectFinalized(ctx, sel, fi, typeMap)
}

func filterSelectFinalized(ctx context.Context, sel sq.SelectBuilder, fi *persistence.FilterInfo, tm map[string]string) (sq.SelectBuilder, error) {
	fop, err := filterOp(ctx, fi, tm)
	if err != nil {
		return sel, err
	}
	return sel.Where(fop), nil
}

func escapeLike(value persistence.FilterSerialization) string {
	v, _ := value.Value()
	vs, _ := v.(string)
	vs = strings.ReplaceAll(vs, "[", "[[]")
	vs = strings.ReplaceAll(vs, "%", "[%]")
	vs = strings.ReplaceAll(vs, "_", "[_]")
	return vs
}

func mapField(f string, tm map[string]string) string {
	if tm == nil {
		return f
	}
	if mf, ok := tm[f]; ok {
		return mf
	}
	return f
}

func filterOp(ctx context.Context, op *persistence.FilterInfo, tm map[string]string) (sq.Sqlizer, error) {
	switch op.Op {
	case persistence.FilterOpOr:
		return filterOr(ctx, op, tm)
	case persistence.FilterOpAnd:
		return filterAnd(ctx, op, tm)
	case persistence.FilterOpEq:
		return sq.Eq{mapField(op.Field, tm): op.Value}, nil
	case persistence.FilterOpNe:
		return sq.NotEq{mapField(op.Field, tm): op.Value}, nil
	case persistence.FilterOpCont:
		return sq.Like{mapField(op.Field, tm): fmt.Sprintf("%%%s%%", escapeLike(op.Value))}, nil
	case persistence.FilterOpNotCont:
		return sq.NotLike{mapField(op.Field, tm): fmt.Sprintf("%%%s%%", escapeLike(op.Value))}, nil
	case persistence.FilterOpICont:
		return sq.ILike{mapField(op.Field, tm): fmt.Sprintf("%%%s%%", escapeLike(op.Value))}, nil
	case persistence.FilterOpNotICont:
		return sq.NotILike{mapField(op.Field, tm): fmt.Sprintf("%%%s%%", escapeLike(op.Value))}, nil
	case persistence.FilterOpGt:
		return sq.Gt{mapField(op.Field, tm): op.Value}, nil
	case persistence.FilterOpGte:
		return sq.GtOrEq{mapField(op.Field, tm): op.Value}, nil
	case persistence.FilterOpLt:
		return sq.Lt{mapField(op.Field, tm): op.Value}, nil
	case persistence.FilterOpLte:
		return sq.LtOrEq{mapField(op.Field, tm): op.Value}, nil
	default:
		return nil, i18n.NewError(ctx, i18n.MsgUnsupportedSQLOpInFilter, op.Op)
	}
}

func filterOr(ctx context.Context, op *persistence.FilterInfo, tm map[string]string) (sq.Sqlizer, error) {
	var err error
	or := make(sq.Or, len(op.Children))
	for i, c := range op.Children {
		if or[i], err = filterOp(ctx, c, tm); err != nil {
			return nil, err
		}
	}
	return or, nil
}

func filterAnd(ctx context.Context, op *persistence.FilterInfo, tm map[string]string) (sq.Sqlizer, error) {
	var err error
	and := make(sq.And, len(op.Children))
	for i, c := range op.Children {
		if and[i], err = filterOp(ctx, c, tm); err != nil {
			return nil, err
		}
	}
	return and, nil
}
