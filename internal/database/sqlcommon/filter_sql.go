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
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/i18n"
)

func (s *SQLCommon) filterSelect(ctx context.Context, sel sq.SelectBuilder, filter database.Filter, typeMap map[string]string) (sq.SelectBuilder, error) {
	fi, err := filter.Finalize()
	if err != nil {
		return sel, err
	}
	if len(fi.Sort) == 0 {
		fi.Sort = []string{s.options.SequenceField}
		fi.Descending = true
	}
	sel, err = s.filterSelectFinalized(ctx, sel, fi, typeMap)
	direction := ""
	if fi.Descending {
		direction = " DESC"
	}
	sort := make([]string, len(fi.Sort))
	for i, field := range fi.Sort {
		sort[i] = s.mapField(field, typeMap)
	}
	sel = sel.OrderBy(fmt.Sprintf("%s%s", strings.Join(sort, ","), direction))
	if err == nil {
		if fi.Skip > 0 {
			sel = sel.Offset(fi.Skip)
		}
		if fi.Limit > 0 {
			sel = sel.Limit(fi.Limit)
		}
	}
	return sel, err
}

func (s *SQLCommon) filterSelectFinalized(ctx context.Context, sel sq.SelectBuilder, fi *database.FilterInfo, tm map[string]string) (sq.SelectBuilder, error) {
	fop, err := s.filterOp(ctx, fi, tm)
	if err != nil {
		return sel, err
	}
	return sel.Where(fop), nil
}

func (s *SQLCommon) buildUpdate(ctx context.Context, sel sq.UpdateBuilder, update database.Update, typeMap map[string]string) (sq.UpdateBuilder, error) {
	ui, err := update.Finalize()
	if err != nil {
		return sel, err
	}
	for _, so := range ui.SetOperations {
		sel = sel.Set(s.mapField(so.Field, typeMap), so.Value)
	}
	return sel, nil
}

func (s *SQLCommon) filterUpdate(ctx context.Context, update sq.UpdateBuilder, filter database.Filter, typeMap map[string]string) (sq.UpdateBuilder, error) {
	fi, err := filter.Finalize()
	var fop sq.Sqlizer
	if err == nil {
		fop, err = s.filterOp(ctx, fi, typeMap)
	}
	if err != nil {
		return update, err
	}
	return update.Where(fop), nil
}

func (s *SQLCommon) escapeLike(value database.FieldSerialization) string {
	v, _ := value.Value()
	vs, _ := v.(string)
	vs = strings.ReplaceAll(vs, "[", "[[]")
	vs = strings.ReplaceAll(vs, "%", "[%]")
	vs = strings.ReplaceAll(vs, "_", "[_]")
	return vs
}

func (s *SQLCommon) mapField(f string, tm map[string]string) string {
	if f == "sequence" {
		return s.options.SequenceField
	}
	if tm == nil {
		return f
	}
	if mf, ok := tm[f]; ok {
		return mf
	}
	return f
}

func (s *SQLCommon) filterOp(ctx context.Context, op *database.FilterInfo, tm map[string]string) (sq.Sqlizer, error) {
	switch op.Op {
	case database.FilterOpOr:
		return s.filterOr(ctx, op, tm)
	case database.FilterOpAnd:
		return s.filterAnd(ctx, op, tm)
	case database.FilterOpEq:
		return sq.Eq{s.mapField(op.Field, tm): op.Value}, nil
	case database.FilterOpNe:
		return sq.NotEq{s.mapField(op.Field, tm): op.Value}, nil
	case database.FilterOpCont:
		return sq.Like{s.mapField(op.Field, tm): fmt.Sprintf("%%%s%%", s.escapeLike(op.Value))}, nil
	case database.FilterOpNotCont:
		return sq.NotLike{s.mapField(op.Field, tm): fmt.Sprintf("%%%s%%", s.escapeLike(op.Value))}, nil
	case database.FilterOpICont:
		return sq.ILike{s.mapField(op.Field, tm): fmt.Sprintf("%%%s%%", s.escapeLike(op.Value))}, nil
	case database.FilterOpNotICont:
		return sq.NotILike{s.mapField(op.Field, tm): fmt.Sprintf("%%%s%%", s.escapeLike(op.Value))}, nil
	case database.FilterOpGt:
		return sq.Gt{s.mapField(op.Field, tm): op.Value}, nil
	case database.FilterOpGte:
		return sq.GtOrEq{s.mapField(op.Field, tm): op.Value}, nil
	case database.FilterOpLt:
		return sq.Lt{s.mapField(op.Field, tm): op.Value}, nil
	case database.FilterOpLte:
		return sq.LtOrEq{s.mapField(op.Field, tm): op.Value}, nil
	default:
		return nil, i18n.NewError(ctx, i18n.MsgUnsupportedSQLOpInFilter, op.Op)
	}
}

func (s *SQLCommon) filterOr(ctx context.Context, op *database.FilterInfo, tm map[string]string) (sq.Sqlizer, error) {
	var err error
	or := make(sq.Or, len(op.Children))
	for i, c := range op.Children {
		if or[i], err = s.filterOp(ctx, c, tm); err != nil {
			return nil, err
		}
	}
	return or, nil
}

func (s *SQLCommon) filterAnd(ctx context.Context, op *database.FilterInfo, tm map[string]string) (sq.Sqlizer, error) {
	var err error
	and := make(sq.And, len(op.Children))
	for i, c := range op.Children {
		if and[i], err = s.filterOp(ctx, c, tm); err != nil {
			return nil, err
		}
	}
	return and, nil
}
