package jsonengine

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	jsonvalue "github.com/Andrew-M-C/go.jsonvalue"
)

// ----------------
// MARK: type - Condition

// Condition 表示一个条件
type Condition struct {
	Expr

	OR  OR  `json:"or,omitempty"  yaml:"or,omitempty"`
	AND AND `json:"and,omitempty" yaml:"and,omitempty"`

	NOT *NOT `json:"not,omitempty" yaml:"not,omitempty"`
}

type conditionWrapping Condition

// UnmarshalJSON is redefined to support simple SQL style expression written as
// a 3-element-array, like ["result.list.[+].code", "=", 200]
func (c *Condition) UnmarshalJSON(b []byte) error {
	w := &conditionWrapping{}
	officialErr := json.Unmarshal(b, w)
	if officialErr == nil {
		*c = *(*Condition)(w)
		return nil
	}

	j, err := jsonvalue.Unmarshal(b)
	if err != nil {
		return officialErr
	}
	if !j.IsArray() {
		return officialErr
	}
	if j.Len() != 3 {
		return fmt.Errorf("SQL style expr should have length 3, but got %d", j.Len())
	}

	f, err := j.GetString(0)
	if err != nil {
		return fmt.Errorf("get SQL style expr field error (%w)", err)
	}

	o, err := j.GetString(1)
	if err != nil {
		return fmt.Errorf("get SQL style expr operator error (%w)", err)
	}

	c.Field = f
	c.Operator = o
	c.Value, _ = j.Get(2)
	return nil
}

// ----------------
// MARK: type - Expr

// Expr 表示一个最简单的表达式条件。
//
// Field 使用点分隔, 需要注意的是, [*] 表示数组中所有的类型都需要匹配, [+] 表示数组中任意一个满足条件即可
type Expr struct {
	Field    string `json:"field,omitempty" yaml:"field,omitempty"`
	Operator string `json:"op"              yaml:"op"`
	Value    any    `json:"value"           yaml:"value"`

	// lazy init
	targetValue *jsonvalue.V
	fieldChain  []field
}

func (e *Expr) match(v *jsonvalue.V) (bool, error) {
	if e.fieldChain == nil {
		e.fieldChain = parseField(e.Field)
	}
	if e.targetValue == nil {
		tgt, err := jsonvalue.Import(e.Value)
		if err != nil {
			return false, fmt.Errorf("%w (%v)", ErrImportTargetValue, err)
		}
		e.targetValue = tgt
	}

	debug("got expr %+v", e)

	// 当前值比较
	if len(e.fieldChain) == 0 {
		return compare(v, e.Operator, e.targetValue)
	}

	// 以下层层匹配
	subExpr := &Expr{
		Operator:    e.Operator,
		targetValue: e.targetValue,
		fieldChain:  e.fieldChain[1:],
	}

	// object 就是最简单的单层匹配就行了
	top := e.fieldChain[0]
	if top.Object != "" {
		subV, err := v.Get(top.Object)
		if err != nil {
			debug("Get and got error: '%v', top field '%v', value %v", err, top.Object, v)
			return false, err
		}
		return subExpr.match(subV)
	}

	// 以下是数组逻辑
	if !v.IsArray() {
		return false, fmt.Errorf("%w, target to match is not an array", ErrTypeNotMatch)
	}

	// 数组中的任意一个
	if top.Array.Any {
		var lastErr error
		for _, subV := range v.ForRangeArr() {
			b, err := subExpr.match(subV)
			if err != nil {
				lastErr = err
				continue
			}
			if b {
				// 只要有一个符合条件, 那么就不返回 err 了
				return true, nil
			}
		}
		return false, lastErr
	}

	// 数组中的每一个
	if top.Array.All {
		for _, subV := range v.ForRangeArr() {
			b, err := subExpr.match(subV)
			if err != nil {
				return false, err
			}
			if !b {
				return false, nil
			}
		}
		return true, nil
	}

	// 如果是指定 array 的具体某个 index, 那也算简单匹配
	subV, err := v.Get(top.Array.At)
	if err != nil {
		return false, err
	}
	return subExpr.match(subV)
}

func compare(v *jsonvalue.V, op string, target *jsonvalue.V) (bool, error) {
	switch op = strings.ToLower(strings.TrimSpace(op)); op {
	default:
		// 所有奇怪的符号都交给这里统一过滤
		return compareNumber(v, op, target)

	case "≹", "≸", "≠", "<>", "!=", "ne":
		res := !v.Equal(target)
		debug("%v != %v ? %v", v, target, res)
		return res, nil

	case "=", "==", "===", "eq":
		if v.ValueType() != target.ValueType() {
			return false, fmt.Errorf(
				"%w value and target should have same value type, but got (%v, %v)",
				ErrTypeNotMatch, v.ValueType(), target.ValueType(),
			)
		}
		res := v.Equal(target)
		debug("%v == %v ? %v", v, target, res)
		return res, nil

	case "in":
		return compareIn(v, target)
	}
}

func compareNumber(v *jsonvalue.V, op string, target *jsonvalue.V) (bool, error) {
	if v.IsNumber() && target.IsNumber() {
		// OK
	} else {
		return false, fmt.Errorf(
			"%w, expected both value and target both number, but got (%v, %v)",
			ErrTypeNotMatch, v.ValueType(), target.ValueType(),
		)
	}

	switch op {
	default:
		return false, fmt.Errorf("%w (%s)", ErrIllegalOperator, op)

	case "≶", "≷":
		res := !v.Equal(target)
		debug("%v != %v ? %v", v, target, res)
		return res, nil

	case "<", "≱", "lt":
		res := v.Float64() < target.Float64()
		debug("%v < %v ? %v", v, target, res)
		return res, nil

	case "<=", "≤", "≦", "≯", "le":
		res := v.Float64() <= target.Float64()
		debug("%v <= %v ? %v", v, target, res)
		return res, nil

	case ">", "≰", "gt":
		res := v.Float64() > target.Float64()
		debug("%v > %v ? %v", v, target, res)
		return res, nil

	case ">=", "≥", "≧", "≮", "ge":
		res := v.Float64() >= target.Float64()
		debug("%v >= %v ? %v", v, target, res)
		return res, nil
	}
}

func compareIn(v *jsonvalue.V, target *jsonvalue.V) (bool, error) {
	if !target.IsArray() {
		return false, fmt.Errorf(
			"%w, target value should be an array but got %v",
			ErrTypeNotMatch, target.ValueType(),
		)
	}

	for _, subTarget := range target.ForRangeArr() {
		if v.Equal(subTarget) {
			return true, nil
		}
	}

	return false, nil
}

// ----------------
// MARK: type - OR

// OR 表示几个或条件
type OR []Condition

// ----------------
// MARK: type - AND

// AND 表示几个与条件
type AND []Condition

// ----------------
// MARK: type - NOT

// NOT 表示非
type NOT struct {
	Condition
}

// ----------------
// MARK: type - field

type field struct {
	// 表示 object 类型的一个字段
	Object string
	// 表示数组的字段
	Array struct {
		All bool
		Any bool
		At  int
	}
}

// parseField 解析 Field 字段
func parseField(f string) []field {
	parts := strings.Split(f, ".")
	fieldChain := make([]field, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			if f, ok := parseArrayField(part); ok {
				fieldChain = append(fieldChain, f)
			}
		} else {
			fieldChain = append(fieldChain, field{Object: part})
		}
	}

	return fieldChain
}

func parseArrayField(part string) (field, bool) {
	f := field{}

	le := len(part)
	switch s := part[1 : le-1]; s {
	default:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return f, false
		}
		f.Array.At = int(n)
		return f, true

	case "*":
		f.Array.All = true
		return f, true

	case "+":
		f.Array.Any = true
		return f, true
	}
}
