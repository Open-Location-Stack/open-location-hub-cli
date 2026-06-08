package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

type Printer struct {
	JSON    bool
	NoColor bool
	Out     io.Writer
	Err     io.Writer
}

func New(jsonOut, noColor bool) *Printer {
	return &Printer{
		JSON:    jsonOut,
		NoColor: noColor || os.Getenv("NO_COLOR") != "",
		Out:     os.Stdout,
		Err:     os.Stderr,
	}
}

func (p *Printer) Print(value any) error {
	if p.JSON {
		return p.printJSON(value, false)
	}
	return p.printHuman(value)
}

func (p *Printer) PrintLine(value any) error {
	return p.printJSON(value, true)
}

func (p *Printer) Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if p.NoColor {
		fmt.Fprintln(p.Err, msg)
		return
	}
	fmt.Fprintf(p.Err, "\033[36m%s\033[0m\n", msg)
}

func (p *Printer) Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if p.NoColor {
		fmt.Fprintln(p.Out, msg)
		return
	}
	fmt.Fprintf(p.Out, "\033[32m%s\033[0m\n", msg)
}

func (p *Printer) Errorf(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	if p.NoColor {
		return fmt.Errorf("%s", msg)
	}
	return fmt.Errorf("\033[31m%s\033[0m", msg)
}

func (p *Printer) printJSON(value any, compact bool) error {
	var (
		b   []byte
		err error
	)
	if compact {
		b, err = json.Marshal(value)
	} else {
		b, err = json.MarshalIndent(value, "", "  ")
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(p.Out, string(b))
	return err
}

func (p *Printer) printHuman(value any) error {
	switch v := value.(type) {
	case string:
		_, err := fmt.Fprintln(p.Out, v)
		return err
	case []string:
		for _, item := range v {
			if _, err := fmt.Fprintln(p.Out, item); err != nil {
				return err
			}
		}
		return nil
	}

	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		_, err := fmt.Fprintln(p.Out, "<nil>")
		return err
	}
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		return p.printSlice(rv)
	}
	return p.printJSON(value, false)
}

func (p *Printer) printSlice(rv reflect.Value) error {
	if rv.Len() == 0 {
		_, err := fmt.Fprintln(p.Out, "[]")
		return err
	}
	first := indirect(rv.Index(0))
	if !first.IsValid() || first.Kind() != reflect.Struct {
		return p.printJSON(valueFrom(rv), false)
	}

	headers, rows := structRows(rv)
	tw := table.NewWriter()
	tw.SetOutputMirror(p.Out)
	tw.AppendHeader(headers)
	for _, row := range rows {
		tw.AppendRow(row)
	}
	tw.Render()
	return nil
}

func structRows(rv reflect.Value) (table.Row, []table.Row) {
	fieldOrder := map[string]int{}
	var fieldNames []string
	for i := 0; i < rv.Len(); i++ {
		item := indirect(rv.Index(i))
		if !item.IsValid() || item.Kind() != reflect.Struct {
			continue
		}
		for _, name := range exportedFieldNames(item.Type()) {
			if _, ok := fieldOrder[name]; ok {
				continue
			}
			fieldOrder[name] = len(fieldNames)
			fieldNames = append(fieldNames, name)
		}
	}
	sort.SliceStable(fieldNames, func(i, j int) bool {
		a, b := strings.ToLower(fieldNames[i]), strings.ToLower(fieldNames[j])
		if priority(a) != priority(b) {
			return priority(a) < priority(b)
		}
		return a < b
	})

	header := table.Row{}
	for _, name := range fieldNames {
		header = append(header, name)
	}
	rows := make([]table.Row, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		item := indirect(rv.Index(i))
		row := table.Row{}
		for _, name := range fieldNames {
			row = append(row, renderField(item, name))
		}
		rows = append(rows, row)
	}
	return header, rows
}

func exportedFieldNames(t reflect.Type) []string {
	names := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		names = append(names, field.Name)
	}
	return names
}

func renderField(v reflect.Value, name string) string {
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return ""
	}
	field := v.FieldByName(name)
	if !field.IsValid() {
		return ""
	}
	field = indirect(field)
	if !field.IsValid() {
		return ""
	}
	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Bool:
		if field.Bool() {
			return "true"
		}
		return "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", field.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", field.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%v", field.Float())
	case reflect.Slice, reflect.Map, reflect.Struct, reflect.Array:
		b, _ := json.Marshal(field.Interface())
		return string(b)
	default:
		return fmt.Sprintf("%v", field.Interface())
	}
}

func indirect(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func valueFrom(v reflect.Value) any {
	return v.Interface()
}

func priority(name string) int {
	switch name {
	case "id":
		return 0
	case "name", "type":
		return 1
	default:
		return 2
	}
}
