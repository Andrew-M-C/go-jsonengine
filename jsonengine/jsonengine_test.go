package jsonengine

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	jsonvalue "github.com/Andrew-M-C/go.jsonvalue"
	"github.com/smartystreets/goconvey/convey"
)

var (
	cv = convey.Convey
	so = convey.So
	eq = convey.ShouldEqual

	isNil = convey.ShouldBeNil
	isErr = convey.ShouldBeError
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestJSONEngine(t *testing.T) {
	debug = t.Logf
	cv("Match with simple Expr", t, func() { testJSONEngineMatchExpr(t) })
	cv("Match with OR", t, func() { testJSONEngineMatchOR(t) })
	cv("Match with AND", t, func() { testJSONEngineMatchAND(t) })
	cv("Match with NOT", t, func() { testJSONEngineMatchNOT(t) })
	cv("Match with multiple embedded conditions", t, func() { testJSONEngineMatchWithMultipleEmbedding(t) })
	cv("SQL-style expr", t, func() { testSQLStyleExpr(t) })
}

type testCase struct {
	value     string
	cond      string
	expect    bool
	shouldErr bool
	opts      []Option
}

func iterateTestCases(t *testing.T, testName string, cases []testCase) {
	cv(testName, func() {
		for i, testCase := range cases {
			t.Log(testName, "- No", i+1)
			j, err := jsonvalue.UnmarshalString(testCase.value)
			so(err, isNil)

			cond := Condition{}
			err = json.Unmarshal([]byte(testCase.cond), &cond)
			so(err, isNil)

			match, err := Match(j, cond, testCase.opts...)
			if testCase.shouldErr {
				so(err, isErr)
			} else {
				so(err, isNil)
			}
			so(match, eq, testCase.expect)
		}
	})
}

func testJSONEngineMatchExpr(t *testing.T) {
	iterateTestCases(t, "simple object", []testCase{
		{
			value:  `{"lv_a":{"lv_b":{"str":"Hello, world!"}}}`,
			cond:   `{"field":"lv_a.lv_b.str","op":"=","value":"Hello, world!"}`,
			expect: true,
		},
		{
			value:  `{"lv_a":{"lv_b":{"str":"Hello, world!"}}}`,
			cond:   `{"field":"lv_a.lv_b.str","op":"=","value":"Hello!"}`,
			expect: false,
		},
		{
			value:  `{"lv_a":{"lv_b":{"str":"Hello, world!"}}}`,
			cond:   `{"field":"lv_a.lv_b.str","op":"=","value":"Hello!"}`,
			expect: false,
		},
		{
			value:     `{"lv_a":{"lv_b":{"str":"Hello, world!"}}}`,
			cond:      `{"field":"lv_a.lv_b.int","op":"=","value":12345}`,
			shouldErr: true,
		},
		{
			value:  `{"lv_a":{"lv_b":{"str":"Hello, world!"}}}`,
			cond:   `{"field":"lv_a.lv_b.int","op":"=","value":12345}`,
			opts:   []Option{OptWhenNotFound(ReturnFalse)},
			expect: false,
		},

		{
			value:  `{"lv_a":{"lv_b":{"str":"Hello, world!","int":12345}}}`,
			cond:   `{"field":"lv_a.lv_b.int","op":">","value":10000}`,
			expect: true,
		},
		{
			value:  `{"lv_a":{"lv_b":{"str":"Hello, world!","int":12345}}}`,
			cond:   `{"field":"lv_a.lv_b.int","op":">","value":20000}`,
			expect: false,
		},
		{
			value:  `{"lv_a":{"lv_b":{"str":"Hello, world!","int":12345}}}`,
			cond:   `{"field":"lv_a.lv_b.int","op":"IN","value":[10000, 20000]}`,
			expect: false,
		},
		{
			value:  `{"lv_a":{"lv_b":{"str":"Hello, world!","int":12345}}}`,
			cond:   `{"field":"lv_a.lv_b.int","op":"IN","value":[12345,67890]}`,
			expect: true,
		},
		{
			value:  `{"lv_a":{"lv_b":{"str":"Hello, world!","int":12345}}}`,
			cond:   `{"field":"lv_a.lv_b.int","op":"IN","value":[12345]}`,
			expect: true,
		},
		{
			value:  `{"lv_a":{"lv_b":{"str":"Hello, world!","int":12345}}}`,
			cond:   `{"field":"lv_a.lv_b.int","op":"IN","value":[67890]}`,
			expect: false,
		},
	})

	iterateTestCases(t, "simple array", []testCase{
		{
			value:  `{"array":[{"str":"001"},{"str":"002"},{"str":"003","int":12345}]}`,
			cond:   `{"field":"array.[+].int","op":"=","value":12345}`,
			expect: true,
		},
		{
			value:     `{"array":[{"str":"001"},{"str":"002"},{"str":"003","int":"12345"}]}`,
			cond:      `{"field":"array.[+].int","op":"=","value":12345}`,
			shouldErr: true,
		},
		{
			value:  `{"array":[{"str":"001"},{"str":"002"},{"str":"003","int":"12345"}]}`,
			cond:   `{"field":"array.[+].int","op":"=","value":12345}`,
			opts:   []Option{OptWhenTypeMismatch(ReturnFalse)},
			expect: false,
		},
		{
			value:  `{"array":[{"str":"001"},{"str":"002"},{"str":"003","int":12345}]}`,
			cond:   `{"field":"array.[*].int","op":"=","value":12345}`,
			opts:   []Option{OptWhenNotFound(ReturnFalse)},
			expect: false,
		},
		{
			value:  `{"array":[{"str":"001","int":11},{"str":"002","int":22},{"str":"003","int":33}]}`,
			cond:   `{"field":"array.[*].int","op":">","value":0}`,
			expect: true,
		},
		{
			value:  `{"array":[{"str":"001","int":11},{"str":"002","int":22},{"str":"003","int":33}]}`,
			cond:   `{"field":"array.[*].int","op":"<","value":33}`,
			expect: false,
		},
		{
			value:  `{"array":[{"str":"001","int":11},{"str":"002","int":22},{"str":"003","int":33}]}`,
			cond:   `{"field":"array.[+].int","op":"<","value":33}`,
			expect: true,
		},
		{
			value:  `{"array":[{"str":"001","int":11},{"str":"002","int":22},{"str":"003","int":33}]}`,
			cond:   `{"field":"array.[+].int","op":"IN","value":[11,22,33]}`,
			expect: true,
		},
		{
			value:  `{"array":[{"str":"001","int":11},{"str":"002","int":22},{"str":"003","int":33}]}`,
			cond:   `{"field":"array.[*].int","op":"IN","value":[11,22,33]}`,
			expect: true,
		},
		{
			value:  `{"array":[{"str":"001","int":11},{"str":"002","int":22},{"str":"003","int":33}]}`,
			cond:   `{"field":"array.[*].int","op":"IN","value":[11,22]}`,
			expect: false,
		},
		{
			value:  `{"array":[{"str":"001","int":11},{"str":"002","int":22},{"str":"003","int":33}]}`,
			cond:   `{"field":"array.[+].int","op":"IN","value":[11,22]}`,
			expect: true,
		},
		{
			value:  `{"array":[{"str":"001","int":11},{"str":"002","int":22},{"str":"003","int":33}]}`,
			cond:   `{"field":"array.[0].int","op":"IN","value":[11,22]}`,
			expect: true,
		},
		{
			value:  `{"array":[{"str":"001","int":11},{"str":"002","int":22},{"str":"003","int":33}]}`,
			cond:   `{"field":"array.[-1].int","op":"IN","value":[11,22]}`,
			expect: false,
		},
		{
			value:  `{"array":[{"str":"001","int":11},{"str":"002","int":22},{"str":"003","int":33}]}`,
			cond:   `{"field":"array.[-1].int","op":"IN","value":[33]}`,
			expect: true,
		},
	})
}

func testJSONEngineMatchOR(t *testing.T) {
	iterateTestCases(t, "OR", []testCase{
		{
			value:  `{"lv_a":{"lv_b":{"str":"world"}}}`,
			cond:   `{"or":[{"field":"lv_a.lv_b.str","op":"=","value":"world"}]}`,
			expect: true,
		},
		{
			value:  `{"lv_a":{"lv_b":{"str":"world"}}}`,
			cond:   `{"or":[{"field":"lv_a.lv_b.str","op":"=","value":"hello"}]}`,
			expect: false,
		},
		{
			value:  `{"lv_a":{"lv_b":{"str":"world"}}}`,
			cond:   `{"or":[{"field":"lv_a.lv_b.str","op":"=","value":"hello"},{"field":"lv_a.lv_b.str","op":"=","value":"world"}]}`,
			expect: true,
		},
	})
}

func testJSONEngineMatchAND(t *testing.T) {
	iterateTestCases(t, "AND", []testCase{
		{
			value:  `{"lv_a":{"lv_b":{"str":"world","int":12345}}}`,
			cond:   `{"and":[{"field":"lv_a.lv_b.str","op":"=","value":"world"}]}`,
			expect: true,
		},
		{
			value:  `{"lv_a":{"lv_b":{"str":"world","int":12345}}}`,
			cond:   `{"and":[{"field":"lv_a.lv_b.str","op":"=","value":"world"},{"field":"lv_a.lv_b.int","op":"=","value":12345}]}`,
			expect: true,
		},
		{
			value:  `{"lv_a":{"lv_b":{"str":"world","int":12345}}}`,
			cond:   `{"and":[{"field":"lv_a.lv_b.str","op":"=","value":"world"},{"field":"lv_a.lv_b.int","op":"=","value":1234}]}`,
			expect: false,
		},
	})
}

func testJSONEngineMatchNOT(t *testing.T) {
	iterateTestCases(t, "NOT", []testCase{
		{
			value:  `{"lv_a":{"lv_b":{"str":"world","int":12345}}}`,
			cond:   `{"not":{"and":[{"field":"lv_a.lv_b.str","op":"=","value":"world"}]}}`,
			expect: false,
		},
		{
			value:  `{"lv_a":{"lv_b":{"str":"world","int":12345}}}`,
			cond:   `{"not":{"and":[{"field":"lv_a.lv_b.str","op":"=","value":"world"},{"field":"lv_a.lv_b.int","op":"=","value":12345}]}}`,
			expect: false,
		},
		{
			value:  `{"lv_a":{"lv_b":{"str":"world","int":12345}}}`,
			cond:   `{"not":{"and":[{"field":"lv_a.lv_b.str","op":"=","value":"world"},{"field":"lv_a.lv_b.int","op":"=","value":1234}]}}`,
			expect: true,
		},
	})
}

func testJSONEngineMatchWithMultipleEmbedding(t *testing.T) {
	iterateTestCases(t, "embed", []testCase{
		{
			value:  `{"int":123456,"bool":false,"array":[{"int":11},{"int":22}]}`,
			cond:   `{"or":[{"field":"array.[+].int","op":">","value":10},{"not":{"field":"int","op":">","value":100000}}]}`,
			expect: true,
		},
		{
			value:  `{"int":123456,"bool":false,"array":[{"int":11},{"int":22}]}`,
			cond:   `{"and":[{"field":"array.[+].int","op":">","value":10},{"not":{"field":"int","op":">","value":100000}}]}`,
			expect: false,
		},
	})
}

func testSQLStyleExpr(t *testing.T) {
	iterateTestCases(t, "general", []testCase{
		{
			value:  `{"int":123456,"bool":false,"array":[{"int":11},{"int":22}]}`,
			cond:   `{"or":[{"field":"array.[+].int","op":">","value":10},{"not":{"field":"int","op":">","value":100000}}]}`,
			expect: true,
		},
		{
			value:  `{"int":123456,"bool":false,"array":[{"int":11},{"int":22}]}`,
			cond:   `{"or":[["array.[+].int",">",10],{"not":["int",">",100000]}]}`,
			expect: true,
		},
		{
			value:  `{"int":123456,"bool":false,"array":[{"int":11},{"int":22}]}`,
			cond:   `{"and":[{"field":"array.[+].int","op":">","value":10},{"not":{"field":"int","op":">","value":100000}}]}`,
			expect: false,
		},
		{
			value:  `{"int":123456,"bool":false,"array":[{"int":11},{"int":22}]}`,
			cond:   `{"and":[["array.[+].int",">",10],{"not":["int",">",100000]}]}`,
			expect: false,
		},
		{
			value:  `{"int":123456,"bool":false,"array":[{"int":11},{"int":22}]}`,
			cond:   `{"and":[["array.[+].int",">",10],["bool","=",false]]}`,
			expect: true,
		},
		{
			value:  `{"int":123456,"bool":false,"array":[{"int":11},{"int":22}]}`,
			cond:   `{"and":[["array.[+].int",">",10],["bool","=",true]]}`,
			expect: false,
		},
		{
			value:  `{"int":123456,"bool":false,"array":[{"int":11},{"int":22}]}`,
			cond:   `{"or":[["array.[+].int",">",10],["bool","=",true]]}`,
			expect: true,
		},
	})

	iterateTestCases(t, "time", []testCase{
		{
			value:  `{"time":"2024-01-01 12:34:56"}`,
			cond:   `["time",">=","2024-01-01 12:34:56"]`,
			opts:   []Option{OptDateTimeFormat(time.DateTime)},
			expect: true,
		},
		{
			value:  `{"time":"2024-01-01 12:34:56"}`,
			cond:   `["time",">","2024-01-01 12:34:56"]`,
			opts:   []Option{OptDateTimeFormat(time.DateTime)},
			expect: false,
		},
		{
			value:  `{"time":"2024-01-01"}`,
			cond:   `["time",">=","2024-01-01"]`,
			opts:   []Option{OptDateTimeFormat(time.DateOnly)},
			expect: true,
		},
		{
			value:  `{"time":"2024-01-01"}`,
			cond:   `["time","<=","2024-01-01"]`,
			opts:   []Option{OptDateTimeFormat(time.DateOnly)},
			expect: true,
		},
		{
			value:  `{"time":"2024-01-01"}`,
			cond:   `["time",">","2024-01-01"]`,
			opts:   []Option{OptDateTimeFormat(time.DateOnly)},
			expect: false,
		},
		{
			value:  `{"time":"2024-01-01"}`,
			cond:   `["time","<","2024-01-01"]`,
			opts:   []Option{OptDateTimeFormat(time.DateOnly)},
			expect: false,
		},
		{
			value:  `{"time":"01-01"}`,
			cond:   `["time",">=","01-01"]`,
			opts:   []Option{OptDateTimeFormat("01-02")},
			expect: true,
		},
		{
			value:  `{"time":"01-01"}`,
			cond:   `["time","<=","01-01"]`,
			opts:   []Option{OptDateTimeFormat("01-02")},
			expect: true,
		},
	})
}
