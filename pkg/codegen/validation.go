package codegen

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"strings"
)

type SchemaType string
type SchemaFormat string
type RuleType string
type RuleFormat string
type RuleNumeric *float64

const (
	TypeString  SchemaType = "string"
	TypeNumber  SchemaType = "number"
	TypeInteger SchemaType = "integer"
	TypeBoolean SchemaType = "boolean"
	TypeArray   SchemaType = "array"
	TypeObject  SchemaType = "object"

	FormatDate     SchemaFormat = "date"
	FormatDateTime SchemaFormat = "date-time"
	FormatPassword SchemaFormat = "password"
	FormatByte     SchemaFormat = "byte"
	FormatBinary   SchemaFormat = "binary"
	FormatEmail    SchemaFormat = "email"
	FormatUuid     SchemaFormat = "uuid"
	FormatUri      SchemaFormat = "uri"
	FormatHostname SchemaFormat = "hostname"
	FormatIPv4     SchemaFormat = "ipv4"
	FormatIPv6     SchemaFormat = "ipv6"
)

var SchemaTypeToRule = map[SchemaType]RuleType{
	TypeNumber:  "numeric",
	TypeInteger: "numeric",
	TypeBoolean: "bool",
}

var SchemaFormatToRule = map[SchemaFormat]RuleFormat{
	FormatDate:     "date",
	FormatDateTime: "date:dd-mm-yyyy H:i:s",
	FormatEmail:    "email",
	FormatUuid:     "uuid",
	FormatUri:      "url",
	FormatIPv4:     "ip_v4",
	FormatIPv6:     "ip_v6",
}

func GenerateValidationRules(sref *openapi3.SchemaRef, required bool) string {
	schema := sref.Value
	rules := []string{}

	if schema.ReadOnly {
		return ""
	}

	if required {
		rules = append(rules, `required`)
	}

	if _, hasType := SchemaTypeToRule[SchemaType(schema.Type)]; hasType != false {
		rules = append(rules, fmt.Sprintf(`%s`, string(SchemaTypeToRule[SchemaType(schema.Type)])))
	}

	if schema.Min != nil {
		rules = append(rules, fmt.Sprintf(`min:%.2f`, *schema.Min))
	}

	if schema.Max != nil {
		rules = append(rules, fmt.Sprintf(`max:%.2f`, *schema.Max))
	}

	if schema.MinLength != 0 {
		rules = append(rules, fmt.Sprintf(`min:%d`, schema.MinLength))
	}

	if schema.MaxLength != nil {
		rules = append(rules, fmt.Sprintf(`max:%d`, *schema.MaxLength))
	}

	if format, hasType := SchemaFormatToRule[SchemaFormat(schema.Format)]; hasType != false {
		rules = append(rules, fmt.Sprintf(`%s`, format))
	}

	return strings.Join(rules, ",")
}
