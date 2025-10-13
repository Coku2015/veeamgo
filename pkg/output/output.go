package output

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"text/tabwriter"
)

// Print renders arbitrary data in the selected format.
func Print(format string, data any) error {
	switch format {
	case "json":
		return printJSON(data)
	default:
		return printTable(data)
	}
}

func printJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

func printTable(data any) error {
	if data == nil {
		fmt.Fprintln(os.Stdout, "No records found.")
		return nil
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			fmt.Fprintln(os.Stdout, "No records found.")
			return nil
		}
		val = val.Elem()
		data = val.Interface()
	}

	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		return printSliceTable(val)
	default:
		records := toRecords(data)
		if len(records) == 0 {
			fmt.Fprintln(os.Stdout, "No records found.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		for _, rec := range records {
			fmt.Fprintf(w, "%s:\t%v\n", rec.key, rec.value)
		}
		if err := w.Flush(); err != nil {
			return fmt.Errorf("flush table: %w", err)
		}
	}
	return nil
}

func printSliceTable(val reflect.Value) error {
	if val.Len() == 0 {
		fmt.Fprintln(os.Stdout, "No records found.")
		return nil
	}

	type row map[string]any
	columns := make([]string, 0)
	columnSeen := make(map[string]struct{})
	rows := make([]row, 0, val.Len())

	for i := 0; i < val.Len(); i++ {
		recs := toRecords(val.Index(i).Interface())
		r := make(row, len(recs))
		for _, rec := range recs {
			r[rec.key] = rec.value
			if _, ok := columnSeen[rec.key]; !ok {
				columnSeen[rec.key] = struct{}{}
				columns = append(columns, rec.key)
			}
		}
		rows = append(rows, r)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "%s\n", strings.Join(columns, "\t"))
	for _, r := range rows {
		values := make([]string, len(columns))
		for idx, col := range columns {
			if v, ok := r[col]; ok {
				values[idx] = fmt.Sprint(v)
			} else {
				values[idx] = ""
			}
		}
		fmt.Fprintf(w, "%s\n", strings.Join(values, "\t"))
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush table: %w", err)
	}
	return nil
}

type record struct {
	key   string
	value any
}

func toRecords(data any) []record {
	if data == nil {
		return nil
	}
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		return toRecords(val.Elem().Interface())
	}
	switch val.Kind() {
	case reflect.Struct:
		records := make([]record, 0, val.NumField())
		typ := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}
			name := field.Name
			if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
				name = strings.Split(tag, ",")[0]
				if name == "" {
					name = field.Name
				}
			}
			records = append(records, record{key: name, value: val.Field(i).Interface()})
		}
		return records
	case reflect.Map:
		iter := val.MapRange()
		records := make([]record, 0, val.Len())
		for iter.Next() {
			key := fmt.Sprint(iter.Key().Interface())
			records = append(records, record{key: key, value: iter.Value().Interface()})
		}
		sort.Slice(records, func(i, j int) bool { return records[i].key < records[j].key })
		return records
	default:
		return []record{{key: "value", value: data}}
	}
}
