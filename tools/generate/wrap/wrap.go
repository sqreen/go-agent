// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"bytes"
	"go/format"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/mxschmitt/golang-combinations"
)

func main() {
	// Use reflect when reflect.MakeInterface exists
	interfaces := []string{"http.Flusher", "http.Pusher", "http.CloseNotifier", "http.Hijacker", "io.ReaderFrom", "io.StringWriter"}
	set := combinations.All(interfaces)
	sort.Slice(set, func(i, j int) bool {
		return len(set[i]) > len(set[j])
	})

	funcMap := template.FuncMap{
		"interfaceName": func(interfaces []string) string {
			var str strings.Builder
			for _, t := range interfaces {
				ix := strings.Index(t, ".")
				if ix == -1 {
					panic("unexpected template name: the type name should include its package qualifier")
				}
				str.WriteString(t[ix+1:])
			}
			return str.String()
		},

		"structName": func(interfaceName string) string {
			var str strings.Builder
			str.WriteString(strings.ToLower(interfaceName[0:1]))
			str.WriteString(interfaceName[1:])
			return str.String()
		},
	}

	var buf bytes.Buffer

	template.Must(template.New("").Funcs(funcMap).Parse(tpl)).Execute(&buf, map[string]interface{}{
		"Imports":         []string{"io", "net/http"},
		"WrappedType":     "http.ResponseWriter",
		"WrappedTypeName": "ResponseWriter",
		"Package":         "sqhttp",
		"Combinations":    set,
	})

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Panic(err)
	}
	os.Stdout.Write(formatted)
}

var tpl = `
package {{ .Package }}

import (
{{- range .Imports }}
	"{{.}}"
{{- end }}
)

type (
{{- range .Combinations }}
	{{- with $interfaceName := interfaceName . }}

	{{ structName $interfaceName }} struct {
		{{ $.WrappedType }}
		{{ $interfaceName }}
	}

	{{ $interfaceName }} interface {
	{{- end }}
	{{- range . }}
		{{.}}
	{{- end }}
	}

{{- end }}
)

func adaptWrapper(wrapper, wrapped {{ .WrappedType }}) {{ .WrappedType }} {
	switch actual := wrapped.(type) {
		{{- range .Combinations }}
		{{ with $interfaceName := interfaceName . }}	
		case {{ $interfaceName }}:
			return {{ structName $interfaceName }}{
				{{ $.WrappedTypeName }}: wrapper,
				{{ $interfaceName }}: actual,
			}
		{{- end }}
		{{- end }}

		default:
			return wrapper
	}
}
`
