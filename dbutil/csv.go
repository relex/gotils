package dbutil

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/relex/gotils/logger"
)

func ToCSV(rows interface{}) []string {
	listType := reflect.TypeOf(rows)
	if listType.Kind() != reflect.Slice {
		logger.Panicf("rows is not a slice: type=%s value=%s", listType, rows)
	}
	listValue := reflect.ValueOf(rows)

	rowType := listType.Elem()
	if rowType.Kind() != reflect.Struct {
		logger.Panicf("rows are not structs: type=%se", rowType)
	}

	csvLines := make([]string, 0, listValue.Len())

	for rowIndex := 0; rowIndex < listValue.Len(); rowIndex++ {

		rowValue := listValue.Index(rowIndex)
		csvFields := make([]string, 0, rowType.NumField())

		for fieldIndex := 0; fieldIndex < rowType.NumField(); fieldIndex++ {
			fieldValue := rowValue.Field(fieldIndex)
			// ignore private fields
			if !fieldValue.CanSet() {
				continue
			}
			csvFields = append(csvFields, fmt.Sprint(fieldValue.Interface()))
		}
		csvLines = append(csvLines, strings.Join(csvFields, ","))
	}

	return csvLines
}
