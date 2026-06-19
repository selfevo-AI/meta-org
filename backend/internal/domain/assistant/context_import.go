package assistant

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type DictionaryImportSource struct {
	SourceType string
	SourceName string
	Content    []byte
	ScopeLevel string
	ModuleKey  string
	VersionKey string
}

func NormalizeDictionaryImport(source DictionaryImportSource) (DictionaryImportModel, error) {
	switch strings.ToLower(strings.TrimSpace(source.SourceType)) {
	case ContextSourceJSON:
		return normalizeJSONDictionary(source)
	case ContextSourceYAML:
		return normalizeYAMLDictionary(source)
	case ContextSourceCSV:
		return normalizeCSVDictionary(source)
	case ContextSourceXLSX:
		return normalizeXLSXDictionary(source)
	default:
		return DictionaryImportModel{}, fmt.Errorf("%w: unsupported dictionary source type", ErrValidation)
	}
}

func normalizeJSONDictionary(source DictionaryImportSource) (DictionaryImportModel, error) {
	var model DictionaryImportModel
	if err := json.Unmarshal(source.Content, &model); err != nil {
		return model, fmt.Errorf("%w: invalid dictionary json", ErrValidation)
	}
	return fillImportDefaults(model, source), nil
}

func normalizeYAMLDictionary(source DictionaryImportSource) (DictionaryImportModel, error) {
	var model DictionaryImportModel
	if err := yaml.Unmarshal(source.Content, &model); err != nil {
		return model, fmt.Errorf("%w: invalid dictionary yaml", ErrValidation)
	}
	return fillImportDefaults(model, source), nil
}

func normalizeCSVDictionary(source DictionaryImportSource) (DictionaryImportModel, error) {
	reader := csv.NewReader(bytes.NewReader(source.Content))
	reader.TrimLeadingSpace = true
	rows, err := reader.ReadAll()
	if err != nil {
		return DictionaryImportModel{}, fmt.Errorf("%w: invalid dictionary csv", ErrValidation)
	}
	if len(rows) < 2 {
		return DictionaryImportModel{}, fmt.Errorf("%w: csv dictionary requires header and one row", ErrValidation)
	}
	header := map[string]int{}
	for i, name := range rows[0] {
		header[strings.TrimSpace(name)] = i
	}
	model := DictionaryImportModel{
		VersionKey: source.VersionKey,
		SourceType: ContextSourceCSV,
		SourceName: source.SourceName,
		ScopeLevel: source.ScopeLevel,
		ModuleKey:  source.ModuleKey,
	}
	entitySeen := map[string]bool{}
	domainSeen := map[string]bool{}
	for _, row := range rows[1:] {
		moduleKey := csvValue(row, header, "module_key")
		entityKey := csvValue(row, header, "entity_key")
		fieldKey := csvValue(row, header, "field_key")
		if moduleKey == "" || entityKey == "" || fieldKey == "" {
			return model, fmt.Errorf("%w: csv rows require module_key, entity_key, and field_key", ErrValidation)
		}
		if !domainSeen[moduleKey] {
			model.Domains = append(model.Domains, ContextBusinessDomainInput{ModuleKey: moduleKey, Name: moduleKey, ScopeLevel: source.ScopeLevel})
			domainSeen[moduleKey] = true
		}
		if !entitySeen[entityKey] {
			model.Entities = append(model.Entities, ContextEntityInput{EntityKey: entityKey, ModuleKey: moduleKey, DisplayName: entityKey})
			entitySeen[entityKey] = true
		}
		model.Fields = append(model.Fields, ContextFieldInput{
			EntityKey:      entityKey,
			FieldKey:       fieldKey,
			DisplayName:    csvValue(row, header, "display_name"),
			DataType:       firstNonEmpty(csvValue(row, header, "data_type"), "string"),
			BaseWeight:     csvFloat(row, header, "base_weight", 1),
			IsFinanceField: csvBool(row, header, "is_finance_field"),
			TableName:      csvValue(row, header, "table_name"),
			ColumnName:     csvValue(row, header, "column_name"),
		})
	}
	return fillImportDefaults(model, source), nil
}

func fillImportDefaults(model DictionaryImportModel, source DictionaryImportSource) DictionaryImportModel {
	if model.SourceType == "" {
		model.SourceType = source.SourceType
	}
	if model.SourceName == "" {
		model.SourceName = source.SourceName
	}
	if model.ScopeLevel == "" {
		model.ScopeLevel = firstNonEmpty(source.ScopeLevel, ContextScopeModule)
	}
	if model.ModuleKey == "" {
		model.ModuleKey = source.ModuleKey
	}
	if model.VersionKey == "" {
		model.VersionKey = source.VersionKey
	}
	return model
}

func csvValue(row []string, header map[string]int, key string) string {
	index, ok := header[key]
	if !ok || index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func csvBool(row []string, header map[string]int, key string) bool {
	value := strings.ToLower(csvValue(row, header, key))
	return value == "true" || value == "1" || value == "yes"
}

func csvFloat(row []string, header map[string]int, key string, fallback float64) float64 {
	value := csvValue(row, header, key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
