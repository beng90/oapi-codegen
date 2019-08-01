package codegen

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/go-playground/validator.v9"
	"regexp"
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
	FormatDate:     "ISO8601",
	FormatDateTime: "ISO8601",
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
		rules = append(rules, fmt.Sprintf(`min=%.0f`, *schema.Min))
	}

	if schema.Max != nil {
		rules = append(rules, fmt.Sprintf(`max=%.0f`, *schema.Max))
	}

	if schema.MinLength != 0 {
		rules = append(rules, fmt.Sprintf(`min=%d`, schema.MinLength))
	}

	if schema.MaxLength != nil {
		rules = append(rules, fmt.Sprintf(`max=%d`, *schema.MaxLength))
	}

	if schema.Enum != nil {
		values := ""
		for _, v := range schema.Enum {
			switch v.(type) {
			case string:
				values += fmt.Sprintf(`%s `, v.(string))
			case float64:
				values += fmt.Sprintf(`%.0f `, v.(float64))
			}
		}

		rules = append(rules, fmt.Sprintf(`oneof=%v`, strings.TrimSpace(values)))
	}

	if format, hasType := SchemaFormatToRule[SchemaFormat(schema.Format)]; hasType != false {
		rules = append(rules, fmt.Sprintf(`%s`, format))
	}

	return strings.Join(rules, ",")
}

func IsISO8601Date(fl validator.FieldLevel) bool {
	ISO8601DateRegexString := "^(-?(?:[1-9][0-9]*)?[0-9]{4})-(1[0-2]|0[1-9])-(3[01]|0[1-9]|[12][0-9])(?:T|\\s)(2[0-3]|[01][0-9]):([0-5][0-9]):([0-5][0-9])?(Z)?$"
	ISO8601DateRegex := regexp.MustCompile(ISO8601DateRegexString)

	return ISO8601DateRegex.MatchString(fl.Field().String())
}
