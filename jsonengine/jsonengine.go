// Package jsonengine 提供基于 jsonvalue 的 JSON 规则引擎
package jsonengine

import (
	"errors"

	jsonvalue "github.com/Andrew-M-C/go.jsonvalue"
)

var debug = func(string, ...any) {}

// Match 规则匹配
func Match(value any, cond Condition, opts ...Option) (bool, error) {
	// 迭代每一个或条件
	if len(cond.OR) > 0 {
		debug("do OR")
		for _, c := range cond.OR {
			b, err := Match(value, c, opts...)
			if err != nil {
				return false, err
			}
			if b {
				return true, nil
			}
		}
		return false, nil
	}

	// 迭代每一个与条件
	if len(cond.AND) > 0 {
		debug("do AND")
		for _, c := range cond.AND {
			b, err := Match(value, c, opts...)
			if err != nil {
				return false, err
			}
			if !b {
				return false, nil
			}
		}
		return true, nil
	}

	// NOT 条件
	if cond.NOT != nil {
		debug("do NOT")
		b, err := Match(value, cond.NOT.Condition, opts...)
		if err != nil {
			return false, err
		}
		return !b, nil
	}

	// 单一 field 检查
	debug("do expr")
	v, err := jsonvalue.Import(value)
	if err != nil {
		return false, err
	}

	o := mergeOptions(opts)

	// debug("options: %+v, expr: %+v", o, cond.Expr)

	b, err := cond.Expr.match(v, exprOption{TimeFormat: o.dateTimeFormat})
	if err != nil {
		// 继续错误类型检查
		debug("got error: '%v'", err)
	} else {
		return b, nil
	}

	if errors.Is(err, ErrNotFound) {
		switch o.whenNotFound {
		case ReturnFalse:
			return false, nil
		case ReturnError:
			return false, err
		default:
			return false, err
		}
	}

	if errors.Is(err, ErrTypeNotMatch) {
		switch o.whenTypeMismatch {
		case ReturnFalse:
			return false, nil
		case ReturnError:
			return false, err
		default:
			return false, err
		}
	}

	return false, err
}
