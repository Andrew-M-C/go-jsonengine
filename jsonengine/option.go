package jsonengine

import "time"

// ReturnType 表示如何返回
type ReturnType uint

const (
	// ReturnError 表示返回 jsonvalue 的错误
	ReturnError ReturnType = iota
	// ReturnFalse 表示返回不匹配
	ReturnFalse
)

// Option 表示可用参数
type Option func(*options)

type options struct {
	whenNotFound     ReturnType
	whenTypeMismatch ReturnType
	dateTimeFormat   string
}

// OptWhenNotFound 表示当查找不到值时, 如何返回
func OptWhenNotFound(typ ReturnType) Option {
	return func(o *options) {
		switch typ {
		default:
			// do nothing
		case ReturnFalse, ReturnError:
			o.whenNotFound = typ
		}
	}
}

// OptWhenTypeMismatch 表示当值类型不匹配时, 如何返回
func OptWhenTypeMismatch(typ ReturnType) Option {
	return func(o *options) {
		switch typ {
		default:
			// do nothing
		case ReturnFalse, ReturnError:
			o.whenTypeMismatch = typ
		}
	}
}

// OptDateTimeFormat 表示时间格式, Go 格式, 用于当目标是 string 的时候, 检查是不是时间
func OptDateTimeFormat(format string) Option {
	s := time.Now().Format(format)
	if _, err := time.Parse(format, s); err != nil {
		debug("illegal time format '%s'", format)
		return func(*options) { /* do nothing */ }
	}
	debug("valid time format '%s'", format)
	return func(o *options) {
		o.dateTimeFormat = format
	}
}

func mergeOptions(opts []Option) *options {
	o := &options{}
	for _, f := range opts {
		if f != nil {
			f(o)
		}
	}
	return o
}
