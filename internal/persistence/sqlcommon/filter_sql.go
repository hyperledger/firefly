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

func filterSelect(ctx context.Context, sel sq.SelectBuilder, filter *persistence.FilterInfo) (sq.SelectBuilder, error) {
	fop, err := filterOp(ctx, filter)
	if err != nil {
		return sq.SelectBuilder{}, err
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

func filterOp(ctx context.Context, op *persistence.FilterInfo) (sq.Sqlizer, error) {
	switch op.Op {
	case persistence.FilterOpOr:
		return filterOr(ctx, op)
	case persistence.FilterOpAnd:
		return filterAnd(ctx, op)
	case persistence.FilterOpEq:
		return sq.Eq{op.Field: op.Value}, nil
	case persistence.FilterOpNe:
		return sq.NotEq{op.Field: op.Value}, nil
	case persistence.FilterOpCont:
		return sq.Like{op.Field: fmt.Sprintf("%%%s%%", escapeLike(op.Value))}, nil
	case persistence.FilterOpNotCont:
		return sq.NotLike{op.Field: fmt.Sprintf("%%%s%%", escapeLike(op.Value))}, nil
	case persistence.FilterOpICont:
		return sq.ILike{op.Field: fmt.Sprintf("%%%s%%", escapeLike(op.Value))}, nil
	case persistence.FilterOpNotICont:
		return sq.NotILike{op.Field: fmt.Sprintf("%%%s%%", escapeLike(op.Value))}, nil
	case persistence.FilterOpGt:
		return sq.Gt{op.Field: op.Value}, nil
	case persistence.FilterOpGte:
		return sq.GtOrEq{op.Field: op.Value}, nil
	case persistence.FilterOpLt:
		return sq.Lt{op.Field: op.Value}, nil
	case persistence.FilterOpLte:
		return sq.LtOrEq{op.Field: op.Value}, nil
	default:
		return nil, i18n.NewError(ctx, i18n.MsgUnsupportedSQLOpInFilter, op.Op)
	}
}

func filterOr(ctx context.Context, op *persistence.FilterInfo) (sq.Sqlizer, error) {
	var err error
	or := make(sq.Or, len(op.Children))
	for i, c := range op.Children {
		if or[i], err = filterOp(ctx, c); err != nil {
			return nil, err
		}
	}
	return or, nil
}

func filterAnd(ctx context.Context, op *persistence.FilterInfo) (sq.Sqlizer, error) {
	var err error
	and := make(sq.And, len(op.Children))
	for i, c := range op.Children {
		if and[i], err = filterOp(ctx, c); err != nil {
			return nil, err
		}
	}
	return and, nil
}
