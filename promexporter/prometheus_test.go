package promexporter_test

import (
	"testing"

	"github.com/relex/gotils/promexporter"
	"github.com/stretchr/testify/assert"
)

func TestStructureReflection(t *testing.T) {

	type privateLabels struct {
		ExportedField          string
		ExportedFieldWithLabel string `label:"exported_field_and_label"`
		privateField           string
		privateFieldWithLabel  string `label:"private_field_and_label"`
	}

	labels := privateLabels{
		ExportedField:          "1",
		ExportedFieldWithLabel: "2",
		privateField:           "3",
		privateFieldWithLabel:  "4",
	}

	assert.Equal(t, promexporter.GetLabelNames(labels), []string{"exported_field", "exported_field_and_label", "private_field", "private_field_and_label"})
	assert.Equal(t, promexporter.GetLabelValues(labels), []string{"1", "2", "3", "4"})
	assert.Contains(t, promexporter.GetMetricText(), `logger_logs_total{component="(root)",level="fatal"} 0`)
}
