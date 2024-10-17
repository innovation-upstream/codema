package targetrenderer

import (
	"strings"
	"text/template"

	goTmpl "text/template"

	"github.com/iancoleman/strcase"
	"github.com/innovation-upstream/codema/internal/config"
	"github.com/innovation-upstream/codema/internal/directive"
	"github.com/pkg/errors"
)

type GoTemplateTargetRenderer struct{}

func (r *GoTemplateTargetRenderer) Render(templateContent string, data interface{}) (string, error) {
	tmpl, err := template.New("").Funcs(templateFuncs()).Parse(templateContent)
	if err != nil {
		return "", errors.WithStack(err)
	}

	var w strings.Builder
	err = tmpl.Execute(&w, data)
	if err != nil {
		return "", errors.WithStack(err)
	}

	result := strings.TrimSpace(w.String())

	return result, nil
}

func (r *GoTemplateTargetRenderer) GetType() TargetRendererType {
	return TargetRendererType_GoTemplate
}

func templateFuncs() goTmpl.FuncMap {
	return goTmpl.FuncMap{
		"protoType":                     mapToProtoType,
		"mapGoType":                     mapGoType,
		"mapGoTypeWithCustomTypePrefix": mapGoTypeWithCustomTypePrefix,
		"toGoModelFieldCase":            toGoModelFieldCase,
		"add":                           func(a, b int) int { return a + b },
		"camelCase":                     strcase.ToCamel,
		"camelCaseCapitalizeID":         camelCaseCapitalizeID,
		"lowerCamelCaseCapitalizeID":    lowerCamelCaseCapitalizeID,
		"lowerCamelCaseNoExceptions":    lowerCamelCaseNoExceptions,
		"camelCaseNoExceptions":         camelCaseNoExceptions,
		"snakecase":                     strcase.ToSnake,
		"lowerCamelCase":                strcase.ToLowerCamel,
		"mapGraphQLType":                mapGraphQLType,
		"mapGraphQLInputType":           mapGraphQLInputType,
		"getGraphqlTypeForField":        getGraphqlTypeForField,
		"getGraphqlNameForField":        getGraphqlNameForField,
		"mapTypescriptType":             mapTypescriptType,
		"fieldHasTag":                   fieldHasTag,
		"isPrimitiveFieldType":          config.IsPrimitiveFieldType,
		"getModelDirective":             getModelDirective,
		"getModelDirectiveList":         getModelDirectiveList,
		"getModelTaggedFieldName":       getModelTaggedFieldName,
		"getFieldDirective":             getFieldDirective,
	}
}

func mapToProtoType(codemaType string) string {
	switch codemaType {
	case "ID", "String":
		return "string"
	case "Int":
		return "int64"
	case "Float":
		return "double"
	case "Boolean":
		return "bool"
	case "DateTime":
		return "google.protobuf.Timestamp"
	default:
		if strings.HasPrefix(codemaType, "[") && strings.HasSuffix(codemaType, "]") {
			return "repeated " + mapToProtoType(codemaType[1:len(codemaType)-1])
		}
		return codemaType // For custom types, use as-is
	}
}

func mapGoType(codemaType string) string {
	switch codemaType {
	case "ID", "String":
		return "string"
	case "Int":
		return "int64"
	case "Float":
		return "float64"
	case "Boolean":
		return "bool"
	case "DateTime":
		return "time.Time"
	default:
		if strings.HasPrefix(codemaType, "[") && strings.HasSuffix(codemaType, "]") {
			return "[]" + mapGoType(codemaType[1:len(codemaType)-1])
		}
		return codemaType // For custom types, use as-is
	}
}

func mapGoTypeWithCustomTypePrefix(codemaType string, customTypePrefix string) string {
	switch codemaType {
	case "ID", "String":
		return "string"
	case "Int":
		return "int64"
	case "Float":
		return "float64"
	case "Boolean":
		return "bool"
	case "DateTime":
		return "time.Time"
	default:
		if strings.HasPrefix(codemaType, "[") && strings.HasSuffix(codemaType, "]") {
			return "[]" + mapGoType(codemaType[1:len(codemaType)-1])
		}
		return customTypePrefix + codemaType
	}
}

func mapGraphQLType(t string) string {
	switch t {
	case "ID", "String":
		return "String"
	case "Int":
		return "Int"
	case "Float":
		return "Float"
	case "Boolean":
		return "Boolean"
	case "DateTime":
		return "Int"
	default:
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			return "[" + mapGraphQLType(t[1:len(t)-1]) + "]"
		}
		return t
	}
}

func mapGraphQLInputType(t string) string {
	switch t {
	case "ID", "String":
		return "String"
	case "Int":
		return "Int"
	case "Float":
		return "Float"
	case "Boolean":
		return "Boolean"
	case "DateTime":
		return "Int"
	default:
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			return "[" + mapGraphQLInputType(t[1:len(t)-1]) + "]"
		}
		return t + "Input"
	}
}

func getGraphqlTypeForField(f config.FieldDefinition) string {
	mask := f.GetDirectiveStringValue(directive.WellKnownDirectiveGraphQLTypeNameMask)
	if mask != "" {
		return mask
	}

	return mapGraphQLType(f.Type)
}

func getGraphqlNameForField(f config.FieldDefinition) string {
	mask := f.GetDirectiveStringValue(directive.WellKnownDirectiveGraphQLFieldNameMask)
	if mask != "" {
		return mask
	}

	return strcase.ToLowerCamel(f.Name)
}

func mapTypescriptType(codemaType string) string {
	switch codemaType {
	case "ID", "String":
		return "string"
	case "Int":
		return "number"
	case "Float":
		return "number"
	case "Boolean":
		return "boolean"
	case "DateTime":
		return "number"
	default:
		return codemaType // For custom types and enums, use as-is
	}
}

func toGoModelFieldCase(fieldName string) string {
	// Convert field name to TitleCase
	titleCaseField := strcase.ToCamel(fieldName)

	// Check if the field name ends with "Id" and change it to "ID"
	if strings.HasSuffix(titleCaseField, "Id") {
		titleCaseField = strings.TrimSuffix(titleCaseField, "Id") + "ID"
	}

	return titleCaseField
}

func camelCaseCapitalizeID(fieldName string) string {
	camelCaseField := strcase.ToSnake(fieldName)

	if strings.HasSuffix(camelCaseField, "_id") {
		// Replace the last occurrence of "id" or "Id" with "ID"
		camelCaseField = strcase.ToCamel(strings.TrimSuffix(camelCaseField, "_id")) + "ID"
	}
	return camelCaseField
}

func lowerCamelCaseCapitalizeID(fieldName string) string {
	camelCaseField := strcase.ToSnake(fieldName)

	if strings.HasSuffix(camelCaseField, "_id") {
		// Replace the last occurrence of "id" or "Id" with "ID"
		camelCaseField = strcase.ToLowerCamel(strings.TrimSuffix(camelCaseField, "_id")) + "ID"
	}
	return camelCaseField
}

func lowerCamelCaseNoExceptions(fieldName string) string {
	camelCaseField := strcase.ToSnake(fieldName)

	if strings.HasSuffix(camelCaseField, "_id") {
		camelCaseField = strcase.ToLowerCamel(strings.TrimSuffix(camelCaseField, "_id")) + "Id"
	}
	return camelCaseField
}

func camelCaseNoExceptions(fieldName string) string {
	camelCaseField := strcase.ToSnake(fieldName)

	if strings.HasSuffix(camelCaseField, "_id") {
		camelCaseField = strcase.ToCamel(strings.TrimSuffix(camelCaseField, "_id")) + "Id"
	}
	return camelCaseField
}

func fieldHasTag(field config.FieldDefinition, tagName string) bool {
	var hasTag bool
	for _, t := range field.Tags {
		if t.Name == tagName {
			hasTag = true
			break
		}
	}

	return hasTag
}

func getModelDirective(f config.ModelDefinition, s string, defaultVal string) string {
	val := f.GetDirectiveStringValue(s)
	if val != "" {
		return val
	}

	return defaultVal
}

func getModelDirectiveList(f config.ModelDefinition, s string) []interface{} {
	arrVal := f.GetDirectiveListValue(s)
	if arrVal != nil && len(arrVal) > 0 {
		return arrVal
	}

	return make([]interface{}, 0)
}

func getModelTaggedFieldName(m config.ModelDefinition, tagName string, defaultVal string) string {
	for _, field := range m.Fields {
		for _, fieldTag := range field.Tags {
			if fieldTag.Name == tagName {
				return field.Name
			}
		}
	}

	return defaultVal
}

func getFieldDirective(f config.FieldDefinition, s string, defaultVal string) string {
	val := f.GetDirectiveStringValue(s)
	if val != "" {
		return val
	}

	return defaultVal
}
