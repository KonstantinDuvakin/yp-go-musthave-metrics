package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

// parseStruct парсит сниппет с одной структурой T и возвращает её TypeSpec и StructType.
func parseStruct(t *testing.T, body string) (*ast.TypeSpec, *ast.StructType) {
	t.Helper()
	src := "package p\n\ntype T struct {\n" + body + "\n}\n"
	f, err := parser.ParseFile(token.NewFileSet(), "t.go", src, parser.ParseComments)
	require.NoError(t, err)

	ts := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	return ts, ts.Type.(*ast.StructType)
}

// fmtStmts нормализует форматирование операторов через gofmt,
// чтобы сравнение не зависело от табов и пробелов.
func fmtStmts(t *testing.T, stmts string) string {
	t.Helper()
	src, err := format.Source([]byte("package p\n\nfunc f() {\n" + stmts + "\n}\n"))
	require.NoError(t, err, "сгенерированный код не парсится:\n%s", stmts)
	return string(src)
}

func TestHasResetMarker(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want bool
	}{
		{"точное совпадение", "// generate:reset", true},
		{"без пробела после //", "//generate:reset", true},
		{"лишние пробелы", "//   generate:reset   ", true},
		{"среди других строк", "// Doc.\n// generate:reset\n// more", true},
		{"похожий текст", "// generate:resetter", false},
		{"внутри предложения", "// см. generate:reset в README", false},
		{"без маркера", "// просто комментарий", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "package p\n\n" + tt.doc + "\ntype T struct{}\n"
			f, err := parser.ParseFile(token.NewFileSet(), "t.go", src, parser.ParseComments)
			require.NoError(t, err)

			doc := f.Decls[0].(*ast.GenDecl).Doc
			require.Equal(t, tt.want, hasResetMarker(doc))
		})
	}

	t.Run("nil группа", func(t *testing.T) {
		require.False(t, hasResetMarker(nil))
	})
}

func TestCreateFieldValue(t *testing.T) {
	tests := []struct {
		name  string
		field string
		want  string
	}{
		// примитивы
		{"int", "i int", "s.i = 0"},
		{"int64", "i int64", "s.i = 0"},
		{"float64", "f float64", "s.f = 0"},
		{"byte", "b byte", "s.b = 0"},
		{"string", "str string", `s.str = ""`},
		{"bool", "ok bool", "s.ok = false"},

		// коллекции
		{"слайс", "ints []int", "s.ints = s.ints[:0]"},
		{"слайс указателей", "ptrs []*T", "s.ptrs = s.ptrs[:0]"},
		{"массив", "arr [3]string", "s.arr = [3]string{}"},
		{"массив указателей", "arr [3]*T", "s.arr = [3]*T{}"},
		{"мапа", "m map[string]int", "clear(s.m)"},

		// указатели на примитивы
		{"*int", "p *int", "if s.p != nil {\n*s.p = 0\n}"},
		{"*string", "p *string", "if s.p != nil {\n*s.p = \"\"\n}"},
		{"*bool", "p *bool", "if s.p != nil {\n*s.p = false\n}"},

		// именованные типы — через type assertion
		{"структура по значению", "n Nested",
			"if r, ok := any(&s.n).(interface{ Reset() }); ok {\nr.Reset()\n}"},
		{"указатель на структуру", "n *Nested",
			"if r, ok := any(s.n).(interface{ Reset() }); ok && s.n != nil {\nr.Reset()\n}"},
		{"тип из пакета", "d time.Duration",
			"if r, ok := any(&s.d).(interface{ Reset() }); ok {\nr.Reset()\n}"},
		{"указатель на тип из пакета", "w *time.Time",
			"if r, ok := any(s.w).(interface{ Reset() }); ok && s.w != nil {\nr.Reset()\n}"},
		{"анонимная структура", "a struct{ x int }",
			"if r, ok := any(&s.a).(interface{ Reset() }); ok {\nr.Reset()\n}"},
		{"интерфейс", "v interface{ Reset() }",
			"if r, ok := s.v.(interface{ Reset() }); ok {\nr.Reset()\n}"},

		// nil-типы
		{"func", "fn func()", "s.fn = nil"},
		{"chan", "ch chan int", "s.ch = nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, st := parseStruct(t, tt.field)
			field := st.Fields.List[0]

			got := createFieldValue(field, field.Names[0].Name, "s")
			require.Equal(t, fmtStmts(t, tt.want), fmtStmts(t, got.Value))
		})
	}
}

func TestCreateStruct(t *testing.T) {
	t.Run("несколько имён в одном поле", func(t *testing.T) {
		ts, st := parseStruct(t, "x, y int")
		got := createStruct(ts, st)

		require.Len(t, got.Fields, 2)
		require.Equal(t, "s.x = 0", got.Fields[0].Value)
		require.Equal(t, "s.y = 0", got.Fields[1].Value)
	})

	t.Run("embedded по значению", func(t *testing.T) {
		ts, st := parseStruct(t, "Nested")
		got := createStruct(ts, st)

		require.Len(t, got.Fields, 1)
		require.Contains(t, got.Fields[0].Value, "any(&s.Nested)")
	})

	t.Run("embedded указатель", func(t *testing.T) {
		ts, st := parseStruct(t, "*Nested")
		got := createStruct(ts, st)

		require.Len(t, got.Fields, 1)
		require.Contains(t, got.Fields[0].Value, "any(s.Nested)")
		require.Contains(t, got.Fields[0].Value, "s.Nested != nil")
	})

	t.Run("embedded из пакета", func(t *testing.T) {
		ts, st := parseStruct(t, "*bytes.Buffer")
		got := createStruct(ts, st)

		require.Len(t, got.Fields, 1)
		require.Contains(t, got.Fields[0].Value, "s.Buffer")
	})

	t.Run("имя и ресивер", func(t *testing.T) {
		ts, st := parseStruct(t, "i int")
		got := createStruct(ts, st)

		require.Equal(t, "T", got.Name)
		require.Equal(t, "s", got.ShortName)
	})
}

func TestGetEmbeddedName(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		want string
	}{
		{"Ident", "Nested", "Nested"},
		{"*Ident", "*Nested", "Nested"},
		{"Selector", "time.Time", "Time"},
		{"*Selector", "*time.Time", "Time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, st := parseStruct(t, tt.typ)
			require.Equal(t, tt.want, getEmbeddedName(st.Fields.List[0].Type))
		})
	}

	// embedded-полем может быть только именованный тип, так что []int
	// в структуре не распарсится — проверяем на узле напрямую.
	t.Run("не именованный тип", func(t *testing.T) {
		require.Equal(t, "", getEmbeddedName(&ast.ArrayType{Elt: ast.NewIdent("int")}))
	})
}

// TestTemplate проверяет, что шаблон целиком даёт валидный и корректно
// отформатированный Go-файл.
func TestTemplate(t *testing.T) {
	file := File{
		Package: "p",
		Structs: []Struct{
			{Name: "A", ShortName: "s", Fields: []Field{{Value: "s.i = 0"}, {Value: `s.str = ""`}}},
			{Name: "B", ShortName: "s", Fields: []Field{{Value: "clear(s.m)"}}},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, tmpl.Execute(&buf, file))

	got, err := format.Source(buf.Bytes())
	require.NoError(t, err, "шаблон дал невалидный код:\n%s", buf.String())

	want := `// Code generated by go generate; DO NOT EDIT.
// This file was generated by reset/main.go

package p

func (s *A) Reset() {
	s.i = 0
	s.str = ""
}

func (s *B) Reset() {
	clear(s.m)
}
`
	require.Equal(t, want, string(got))
}
