// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"net/http"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

func TestJSVirtualMachine(t *testing.T) {
	t.Run("passing strings", func(t *testing.T) {
		str := "sqreen"

		program, err := goja.Compile("test", `function foo(s) { return s === '`+str+`' }`, true)
		require.NoError(t, err)

		vm := goja.New()
		v, err := vm.RunProgram(program)
		require.NoError(t, err)

		v = vm.Get("foo")
		require.NotNil(t, v)

		var foo goja.Callable
		err = vm.ExportTo(v, &foo)
		require.NoError(t, err)
		require.NotNil(t, foo)

		strPtr := &str
		v, err = foo(goja.Undefined(), vm.ToValue(strPtr))
		require.NoError(t, err)
		require.Equal(t, false, v.ToBoolean())

		v, err = foo(goja.Undefined(), vm.ToValue(str))
		require.NoError(t, err)
		require.Equal(t, true, v.ToBoolean())
	})

	t.Run("substr", func(t *testing.T) {
		vm := goja.New()
		v, err := vm.RunString(`"'\xe3'".substr(2,1)`)
		require.NoError(t, err)
		require.Equal(t, "'", v.String())
	})

	t.Run("map type binding", func(t *testing.T) {
		headers := http.Header{
			"k0": []string{},
			"k1": []string{"v1"},
			"k2": []string{"v21", "v22"},
			"k3": []string{"v3"},
		}
		vm := goja.New()
		// As discussed here https://github.com/dop251/goja/issues/134 the key
		// enumeration of Go map types returns the methods of the type and not the
		// map keys.
		jsVal := vm.ToValue(map[string][]string(headers))
		vm.Set("headers", jsVal)
		v, err := vm.RunString(`Object.keys(headers)`)
		require.NoError(t, err)
		result := v.Export()
		require.ElementsMatch(t, []interface{}{"k0", "k1", "k2", "k3"}, result)
	})

	t.Run("accessing a function definition", func(t *testing.T) {
		program, err := goja.Compile("test", `function foo() { return 33 }`, true)
		require.NoError(t, err)

		vm := goja.New()
		v, err := vm.RunProgram(program)
		require.NoError(t, err)

		v = vm.Get("foo")
		require.NotNil(t, v)

		var foo goja.Callable
		err = vm.ExportTo(v, &foo)
		require.NoError(t, err)
		require.NotNil(t, foo)

		v, err = foo(goja.Undefined())
		require.NoError(t, err)
		require.Equal(t, int64(33), v.ToInteger())
	})

	t.Run("js exceptions", func(t *testing.T) {
		program, err := goja.Compile("test", `
function foo(n) {
	if (n <= 0) {
		throw "oops";
	}
  foo(n-1);
}
`, true)
		require.NoError(t, err)

		vm := goja.New()
		v, err := vm.RunProgram(program)
		require.NoError(t, err)

		v = vm.Get("foo")
		require.NotNil(t, v)

		var foo goja.Callable
		err = vm.ExportTo(v, &foo)
		require.NoError(t, err)
		require.NotNil(t, foo)

		v, err = foo(goja.Undefined(), vm.ToValue(10))
		require.Error(t, err)

		ex, ok := err.(*goja.Exception)
		require.True(t, ok)
		require.NotNil(t, ex)
		require.Equal(t, "oops", ex.Value().String())

		t.Log(ex.String())
		t.Log(ex.Error())
		t.Logf("value=%#+v", ex.Value().Export())
	})

	t.Run("exporting a structure", func(t *testing.T) {
		program, err := goja.Compile("test", `function foo() { return { A: 27, B: "hey" }; }`, true)
		require.NoError(t, err)

		vm := goja.New()
		v, err := vm.RunProgram(program)
		require.NoError(t, err)

		v = vm.Get("foo")
		require.NotNil(t, v)

		var foo goja.Callable
		err = vm.ExportTo(v, &foo)
		require.NoError(t, err)
		require.NotNil(t, foo)

		v, err = foo(goja.Undefined())
		require.NoError(t, err)

		type myS struct {
			A int
			B string
		}
		var s myS
		err = vm.ExportTo(v, &s)
		require.NoError(t, err)
		require.Equal(t, myS{A: 27, B: "hey"}, s)
	})

	t.Run("exporting a structure", func(t *testing.T) {
		program, err := goja.Compile("test", `
function foo() {
	return {
		status: "raise",
		record: {
			a: "foo",
			b: 33
		}
	};
}
`, true)
		require.NoError(t, err)

		vm := goja.New()
		v, err := vm.RunProgram(program)
		require.NoError(t, err)

		vm.SetFieldNameMapper(goja.TagFieldNameMapper("goja", false))

		v = vm.Get("foo")
		require.NotNil(t, v)

		var foo goja.Callable
		err = vm.ExportTo(v, &foo)
		require.NoError(t, err)
		require.NotNil(t, foo)

		v, err = foo(goja.Undefined())
		require.NoError(t, err)

		type myS struct {
			Status string                 `goja:"status"`
			Record map[string]interface{} `goja:"record"`
		}
		var s myS
		err = vm.ExportTo(v, &s)
		require.NoError(t, err)
		require.Equal(t, myS{
			Status: "raise",
			Record: map[string]interface{}{
				"a": "foo",
				"b": int64(33),
			},
		}, s)
	})

	t.Run("binding a structure", func(t *testing.T) {
		var s = struct {
			Field1 string `goja:"jsField"`
			Field2 int
		}{
			Field1: "hey",
			Field2: 33,
		}

		vm := goja.New()

		vm.SetFieldNameMapper(goja.TagFieldNameMapper("goja", false))
		vm.Set("s", s)

		res, err := vm.RunString("Object.keys(s)")
		require.NoError(t, err)
		// Field2 has no goja field tag and therefore has been hidden by the
		// field name mapper
		require.Equal(t, []interface{}{"jsField"}, res.Export())

		vm.SetFieldNameMapper(nil)
		vm.Set("s", s)
		res, err = vm.RunString("Object.keys(s)")
		require.NoError(t, err)
		require.ElementsMatch(t, []interface{}{"Field1", "Field2"}, res.Export())
	})
}
