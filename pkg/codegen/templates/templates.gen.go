package templates

import "text/template"

var templates = map[string]string{"additional-properties.tmpl": `{{range .Types}}{{$addType := .Schema.AdditionalPropertiesType.TypeDecl}}

// Getter for additional properties for {{.TypeName}}. Returns the specified
// element and whether it was found
func (a {{.TypeName}}) Get(fieldName string) (value {{$addType}}, found bool) {
    if a.AdditionalProperties != nil {
        value, found = a.AdditionalProperties[fieldName]
    }
    return
}

// Setter for additional properties for {{.TypeName}}
func (a *{{.TypeName}}) Set(fieldName string, value {{$addType}}) {
    if a.AdditionalProperties == nil {
        a.AdditionalProperties = make(map[string]{{$addType}})
    }
    a.AdditionalProperties[fieldName] = value
}

// Override default JSON handling for {{.TypeName}} to handle AdditionalProperties
func (a *{{.TypeName}}) UnmarshalJSON(b []byte) error {
    object := make(map[string]json.RawMessage)
	err := json.Unmarshal(b, &object)
	if err != nil {
		return err
	}
{{range .Schema.Properties}}
    if raw, found := object["{{.JsonFieldName}}"]; found {
        err = json.Unmarshal(raw, &a.{{.GoFieldName}})
        if err != nil {
            return errors.Wrap(err, "error reading '{{.JsonFieldName}}'")
        }
        delete(object, "{{.JsonFieldName}}")
    }
{{end}}
    if len(object) != 0 {
        a.AdditionalProperties = make(map[string]{{$addType}})
        for fieldName, fieldBuf := range object {
            var fieldVal {{$addType}}
            err := json.Unmarshal(fieldBuf, &fieldVal)
            if err != nil {
                return errors.Wrap(err, fmt.Sprintf("error unmarshaling field %s", fieldName))
            }
            a.AdditionalProperties[fieldName] = fieldVal
        }
    }
	return nil
}

// Override default JSON handling for {{.TypeName}} to handle AdditionalProperties
func (a {{.TypeName}}) MarshalJSON() ([]byte, error) {
    var err error
    object := make(map[string]json.RawMessage)
{{range .Schema.Properties}}
{{if not .Required}}if a.{{.GoFieldName}} != nil { {{end}}
    object["{{.JsonFieldName}}"], err = json.Marshal(a.{{.GoFieldName}})
    if err != nil {
        return nil, errors.Wrap(err, fmt.Sprintf("error marshaling '{{.JsonFieldName}}'"))
    }
{{if not .Required}} }{{end}}
{{end}}
    for fieldName, field := range a.AdditionalProperties {
		object[fieldName], err = json.Marshal(field)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("error marshaling '%s'", fieldName))
		}
	}
	return json.Marshal(object)
}
{{end}}
`,
	"client-with-responses.tmpl": `// ClientWithResponses builds on ClientInterface to offer response payloads
type ClientWithResponses struct {
    ClientInterface
}

// NewClientWithResponses returns a ClientWithResponses with a default Client:
func NewClientWithResponses(server string) *ClientWithResponses {
    return &ClientWithResponses{
        ClientInterface: &Client{
            Client: http.Client{},
            Server: server,
        },
    }
}

// NewClientWithResponsesAndRequestEditorFunc takes in a RequestEditorFn callback function and returns a ClientWithResponses with a default Client:
func NewClientWithResponsesAndRequestEditorFunc(server string, reqEditorFn RequestEditorFn) *ClientWithResponses {
	return &ClientWithResponses{
		ClientInterface: &Client{
			Client: http.Client{},
			Server: server,
			RequestEditor: reqEditorFn,
		},
	}
}


{{range .}}{{$opid := .OperationId}}{{$op := .}}
type {{$opid | lcFirst}}Response struct {
    Body         []byte
	HTTPResponse *http.Response
    {{- range getResponseTypeDefinitions .}}
    {{.TypeName}} *{{.Schema.TypeDecl}}
    {{- end}}
}

// Status returns HTTPResponse.Status
func (r {{$opid | lcFirst}}Response) Status() string {
    if r.HTTPResponse != nil {
        return r.HTTPResponse.Status
    }
    return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r {{$opid | lcFirst}}Response) StatusCode() int {
    if r.HTTPResponse != nil {
        return r.HTTPResponse.StatusCode
    }
    return 0
}
{{end}}


{{range .}}
{{$opid := .OperationId -}}
{{/* Generate client methods (with responses)*/}}

// {{$opid}}{{if .HasBody}}WithBody{{end}}WithResponse request{{if .HasBody}} with arbitrary body{{end}} returning *{{$opid}}Response
func (c *ClientWithResponses) {{$opid}}{{if .HasBody}}WithBody{{end}}WithResponse(ctx context.Context{{genParamArgs .PathParams}}{{if .RequiresParamObject}}, params *{{$opid}}Params{{end}}{{if .HasBody}}, contentType string, body io.Reader{{end}}) (*{{genResponseTypeName $opid}}, error){
    rsp, err := c.{{$opid}}{{if .HasBody}}WithBody{{end}}(ctx{{genParamNames .PathParams}}{{if .RequiresParamObject}}, params{{end}}{{if .HasBody}}, contentType, body{{end}})
    if err != nil {
        return nil, err
    }
    return Parse{{genResponseTypeName $opid}}(rsp)
}

{{$hasParams := .RequiresParamObject -}}
{{$pathParams := .PathParams -}}
{{$bodyRequired := .BodyRequired -}}
{{range .Bodies}}
func (c *ClientWithResponses) {{$opid}}{{.Suffix}}WithResponse(ctx context.Context{{genParamArgs $pathParams}}{{if $hasParams}}, params *{{$opid}}Params{{end}}, body {{.TypeDef}}) (*{{genResponseTypeName $opid}}, error) {
    rsp, err := c.{{$opid}}{{.Suffix}}(ctx{{genParamNames $pathParams}}{{if $hasParams}}, params{{end}}, body)
    if err != nil {
        return nil, err
    }
    return Parse{{genResponseTypeName $opid}}(rsp)
}
{{end}}

{{end}}{{/* operations */}}

{{/* Generate parse functions for responses*/}}
{{range .}}{{$opid := .OperationId}}

// Parse{{genResponseTypeName $opid}} parses an HTTP response from a {{$opid}}WithResponse call
func Parse{{genResponseTypeName $opid}}(rsp *http.Response) (*{{genResponseTypeName $opid}}, error) {
    bodyBytes, err := ioutil.ReadAll(rsp.Body)
    defer rsp.Body.Close()
    if err != nil {
        return nil, err
    }

    response := {{genResponsePayload $opid}}

    {{genResponseUnmarshal $opid .Spec.Responses}}

    return response, nil
}
{{end}}{{/* range . $opid := .OperationId */}}

`,
	"client.tmpl": `// RequestEditorFn  is the function signature for the RequestEditor callback function
type RequestEditorFn func(req *http.Request, ctx context.Context) error

// Client which conforms to the OpenAPI3 specification for this service.
type Client struct {
    // The endpoint of the server conforming to this interface, with scheme,
    // https://api.deepmap.com for example.
    Server string

    // HTTP client with any customized settings, such as certificate chains.
    Client http.Client

    // A callback for modifying requests which are generated before sending over
    // the network.
    RequestEditor RequestEditorFn
}

// The interface specification for the client above.
type ClientInterface interface {
{{range . -}}
{{$hasParams := .RequiresParamObject -}}
{{$pathParams := .PathParams -}}
{{$opid := .OperationId -}}
    // {{$opid}} request {{if .HasBody}} with any body{{end}}
    {{$opid}}{{if .HasBody}}WithBody{{end}}(ctx context.Context{{genParamArgs $pathParams}}{{if $hasParams}}, params *{{$opid}}Params{{end}}{{if .HasBody}}, contentType string, body io.Reader{{end}}) (*http.Response, error)
{{range .Bodies}}
    {{$opid}}{{.Suffix}}(ctx context.Context{{genParamArgs $pathParams}}{{if $hasParams}}, params *{{$opid}}Params{{end}}, body {{.TypeDef}}) (*http.Response, error)
{{end}}{{/* range .Bodies */}}
{{end}}{{/* range . $opid := .OperationId */}}
}


{{/* Generate client methods */}}
{{range . -}}
{{$hasParams := .RequiresParamObject -}}
{{$pathParams := .PathParams -}}
{{$opid := .OperationId -}}

func (c *Client) {{$opid}}{{if .HasBody}}WithBody{{end}}(ctx context.Context{{genParamArgs $pathParams}}{{if $hasParams}}, params *{{$opid}}Params{{end}}{{if .HasBody}}, contentType string, body io.Reader{{end}}) (*http.Response, error) {
    req, err := New{{$opid}}Request{{if .HasBody}}WithBody{{end}}(c.Server{{genParamNames .PathParams}}{{if $hasParams}}, params{{end}}{{if .HasBody}}, contentType, body{{end}})
    if err != nil {
        return nil, err
    }
    req = req.WithContext(ctx)
    if c.RequestEditor != nil {
        err = c.RequestEditor(req, ctx)
        if err != nil {
            return nil, err
        }
    }
    return c.Client.Do(req)
}

{{range .Bodies}}
func (c *Client) {{$opid}}{{.Suffix}}(ctx context.Context{{genParamArgs $pathParams}}{{if $hasParams}}, params *{{$opid}}Params{{end}}, body {{.TypeDef}}) (*http.Response, error) {
    req, err := New{{$opid}}{{.Suffix}}Request(c.Server{{genParamNames $pathParams}}{{if $hasParams}}, params{{end}}, body)
    if err != nil {
        return nil, err
    }
    req = req.WithContext(ctx)
    if c.RequestEditor != nil {
        err = c.RequestEditor(req, ctx)
        if err != nil {
            return nil, err
        }
    }
    return c.Client.Do(req)
}
{{end}}{{/* range .Bodies */}}
{{end}}

{{/* Generate request builders */}}
{{range .}}
{{$hasParams := .RequiresParamObject -}}
{{$pathParams := .PathParams -}}
{{$bodyRequired := .BodyRequired -}}
{{$opid := .OperationId -}}

{{range .Bodies}}
// New{{$opid}}Request{{.Suffix}} calls the generic {{$opid}} builder with {{.ContentType}} body
func New{{$opid}}Request{{.Suffix}}(server string{{genParamArgs $pathParams}}{{if $hasParams}}, params *{{$opid}}Params{{end}}, body {{.Schema.TypeDecl}}) (*http.Request, error) {
    var bodyReader io.Reader
    buf, err := json.Marshal(body)
    if err != nil {
        return nil, err
    }
    bodyReader = bytes.NewReader(buf)
    return New{{$opid}}RequestWithBody(server{{genParamNames $pathParams}}{{if $hasParams}}, params{{end}}, "{{.ContentType}}", bodyReader)
}
{{end}}

// New{{$opid}}Request{{if .HasBody}}WithBody{{end}} generates requests for {{$opid}}{{if .HasBody}} with any type of body{{end}}
func New{{$opid}}Request{{if .HasBody}}WithBody{{end}}(server string{{genParamArgs $pathParams}}{{if $hasParams}}, params *{{$opid}}Params{{end}}{{if .HasBody}}, contentType string, body io.Reader{{end}}) (*http.Request, error) {
    var err error
{{range $paramIdx, $param := .PathParams}}
    var pathParam{{$paramIdx}} string
    {{if .IsPassThrough}}
    pathParam{{$paramIdx}} = {{.ParamName}}
    {{end}}
    {{if .IsJson}}
    var pathParamBuf{{$paramIdx}} []byte
    pathParamBuf{{$paramIdx}}, err = json.Marshal({{.ParamName}})
    if err != nil {
        return nil, err
    }
    pathParam{{$paramIdx}} = string(pathParamBuf{{$paramIdx}})
    {{end}}
    {{if .IsStyled}}
    pathParam{{$paramIdx}}, err = runtime.StyleParam("{{.Style}}", {{.Explode}}, "{{.ParamName}}", {{.GoVariableName}})
    if err != nil {
        return nil, err
    }
    {{end}}
{{end}}
    queryUrl := fmt.Sprintf("%s{{genParamFmtString .Path}}", server{{range $paramIdx, $param := .PathParams}}, pathParam{{$paramIdx}}{{end}})
{{if .QueryParams}}
    var queryStrings []string
{{range $paramIdx, $param := .QueryParams}}
    var queryParam{{$paramIdx}} string
    {{if not .Required}} if params.{{.GoName}} != nil { {{end}}
    {{if .IsPassThrough}}
    queryParam{{$paramIdx}} = "{{.ParamName}}=" + {{if not .Required}}*{{end}}params.{{.GoName}}
    {{end}}
    {{if .IsJson}}
    var queryParamBuf{{$paramIdx}} []byte
    queryParamBuf{{$paramIdx}}, err = json.Marshal({{if not .Required}}*{{end}}params.{{.GoName}})
    if err != nil {
        return nil, err
    }
    queryParam{{$paramIdx}} = "{{.ParamName}}=" + string(queryParamBuf{{$paramIdx}})

    {{end}}
    {{if .IsStyled}}
    queryParam{{$paramIdx}}, err = runtime.StyleParam("{{.Style}}", {{.Explode}}, "{{.ParamName}}", {{if not .Required}}*{{end}}params.{{.GoName}})
    if err != nil {
        return nil, err
    }
    {{end}}
    queryStrings = append(queryStrings, queryParam{{$paramIdx}})
    {{if not .Required}}}{{end}}
{{end}}
    if len(queryStrings) != 0 {
        queryUrl += "?" + strings.Join(queryStrings, "&")
    }
{{end}}{{/* if .QueryParams */}}
    req, err := http.NewRequest("{{.Method}}", queryUrl, {{if .HasBody}}body{{else}}nil{{end}})
    if err != nil {
        return nil, err
    }

{{range $paramIdx, $param := .HeaderParams}}
    {{if not .Required}} if params.{{.GoName}} != nil { {{end}}
    var headerParam{{$paramIdx}} string
    {{if .IsPassThrough}}
    headerParam{{$paramIdx}} = {{if not .Required}}*{{end}}params.{{.GoName}}
    {{end}}
    {{if .IsJson}}
    var headerParamBuf{{$paramIdx}} []byte
    headerParamBuf{{$paramIdx}}, err = json.Marshal({{if not .Required}}*{{end}}params.{{.GoName}})
    if err != nil {
        return nil, err
    }
    headerParam{{$paramIdx}} = string(headerParamBuf{{$paramIdx}})
    {{end}}
    {{if .IsStyled}}
    headerParam{{$paramIdx}}, err = runtime.StyleParam("{{.Style}}", {{.Explode}}, "{{.ParamName}}", {{if not .Required}}*{{end}}params.{{.GoName}})
    if err != nil {
        return nil, err
    }
    {{end}}
    req.Header.Add("{{.ParamName}}", headerParam{{$paramIdx}})
    {{if not .Required}}}{{end}}
{{end}}

{{range $paramIdx, $param := .CookieParams}}
    {{if not .Required}} if params.{{.GoName}} != nil { {{end}}
    var cookieParam{{$paramIdx}} string
    {{if .IsPassThrough}}
    cookieParam{{$paramIdx}} = {{if not .Required}}*{{end}}params.{{.GoName}}
    {{end}}
    {{if .IsJson}}
    var cookieParamBuf{{$paramIdx}} []byte
    cookieParamBuf{{$paramIdx}}, err = json.Marshal({{if not .Required}}*{{end}}params.{{.GoName}})
    if err != nil {
        return nil, err
    }
    cookieParam{{$paramIdx}} = url.QueryEscape(string(cookieParamBuf{{$paramIdx}}))
    {{end}}
    {{if .IsStyled}}
    cookieParam{{$paramIdx}}, err = runtime.StyleParam("simple", {{.Explode}}, "{{.ParamName}}", {{if not .Required}}*{{end}}params.{{.GoName}})
    if err != nil {
        return nil, err
    }
    {{end}}
    cookie{{$paramIdx}} := &http.Cookie{
        Name:"{{.ParamName}}",
        Value:cookieParam{{$paramIdx}},
    }
    req.AddCookie(cookie{{$paramIdx}})
    {{if not .Required}}}{{end}}
{{end}}
    {{if .HasBody}}req.Header.Add("Content-Type", contentType){{end}}
    return req, nil
}

{{end}}{{/* Range */}}
`,
	"imports.tmpl": `// Package {{.PackageName}} provides primitives to interact the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen DO NOT EDIT.
package {{.PackageName}}

{{if .Imports}}
import (
{{range .Imports}} "{{.}}"
{{end}})
{{end}}
`,
	"inline.tmpl": `// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{
{{range .}}
    "{{.}}",{{end}}
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file.
func GetSwagger() (*openapi3.Swagger, error) {
    zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
    if err != nil {
        return nil, fmt.Errorf("error base64 decoding spec: %s", err)
    }
    zr, err := gzip.NewReader(bytes.NewReader(zipped))
    if err != nil {
        return nil, fmt.Errorf("error decompressing spec: %s", err)
    }
    var buf bytes.Buffer
    _, err = buf.ReadFrom(zr)
    if err != nil {
        return nil, fmt.Errorf("error decompressing spec: %s", err)
    }

    swagger, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData(buf.Bytes())
    if err != nil {
        return nil, fmt.Errorf("error loading Swagger: %s", err)
    }
    return swagger, nil
}
`,
	"param-types.tmpl": `{{range .}}{{$opid := .OperationId}}
{{range .TypeDefinitions}}
// {{.TypeName}} defines parameters for {{$opid}}.
type {{.TypeName}} {{.Schema.TypeDecl}}
{{end}}
{{end}}
`,
	"register.tmpl": `// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router runtime.EchoRouter, si ServerInterface) {
{{if .}}
    wrapper := ServerInterfaceWrapper{
        Handler: si,
    }
{{end}}
{{range .}}router.{{.Method}}("{{.Path | swaggerUriToEchoUri}}", wrapper.{{.OperationId}})
{{end}}
}
`,
	"request-bodies.tmpl": `{{range .}}{{$opid := .OperationId}}
{{range .Bodies}}
// {{$opid}}RequestBody defines body for {{$opid}} for application/json ContentType.
type {{$opid}}{{.NameTag}}RequestBody {{.TypeDef}}
{{end}}
{{end}}
`,
	"server-interface.tmpl": `// ServerInterface represents all server handlers.
type ServerInterface interface {
{{range .}}{{.SummaryAsComment -}}
// ({{.Method}} {{.Path}})
{{.OperationId}}(ctx echo.Context{{genParamArgs .PathParams}}{{if .RequiresParamObject}}, params {{.OperationId}}Params{{end}}) error
{{end}}
}
`,
	"typedef.tmpl": `{{range .Types}}
// {{.TypeName}} defines model for {{.JsonName}}.
type {{.TypeName}} {{.Schema.TypeDecl}}
{{end}}
`,
	"wrappers.tmpl": `// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
    Handler ServerInterface
}

{{range .}}{{$opid := .OperationId}}// {{$opid}} converts echo context to params.
func (w *ServerInterfaceWrapper) {{.OperationId}} (ctx echo.Context) error {
    var err error
{{range .PathParams}}// ------------- Path parameter "{{.ParamName}}" -------------
    var {{$varName := .GoVariableName}}{{$varName}} {{.TypeDef}}
{{if .IsPassThrough}}
    {{$varName}} = ctx.Param("{{.ParamName}}")
{{end}}
{{if .IsJson}}
    err = json.Unmarshal([]byte(ctx.Param("{{.ParamName}}")), &{{$varName}})
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "Error unmarshaling parameter '{{.ParamName}}' as JSON")
    }
{{end}}
{{if .IsStyled}}
    err = runtime.BindStyledParameter("{{.Style}}",{{.Explode}}, "{{.ParamName}}", ctx.Param("{{.ParamName}}"), &{{$varName}})
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter {{.ParamName}}: %s", err))
    }
{{end}}
{{end}}

{{if .RequiresParamObject}}
    // Parameter object where we will unmarshal all parameters from the context
    var params {{.OperationId}}Params
{{range $paramIdx, $param := .QueryParams}}// ------------- {{if .Required}}Required{{else}}Optional{{end}} query parameter "{{.ParamName}}" -------------
    if paramValue := ctx.QueryParam("{{.ParamName}}"); paramValue != "" {
    {{if .IsPassThrough}}
    params.{{.GoName}} = {{if not .Required}}&{{end}}paramValue
    {{end}}
    {{if .IsJson}}
    var value {{.TypeDef}}
    err = json.Unmarshal([]byte(paramValue), &value)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "Error unmarshaling parameter '{{.ParamName}}' as JSON")
    }
    params.{{.GoName}} = {{if not .Required}}&{{end}}value
    {{end}}
    }{{if .Required}} else {
        return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query argument {{.ParamName}} is required, but not found"))
    }{{end}}
    {{if .IsStyled}}
    err = runtime.BindQueryParameter("{{.Style}}", {{.Explode}}, {{.Required}}, "{{.ParamName}}", ctx.QueryParams(), &params.{{.GoName}})
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter {{.ParamName}}: %s", err))
    }
    {{end}}
{{end}}

{{if .HeaderParams}}
    headers := ctx.Request().Header
{{range .HeaderParams}}// ------------- {{if .Required}}Required{{else}}Optional{{end}} header parameter "{{.ParamName}}" -------------
    if valueList, found := headers["{{.ParamName}}"]; found {
        var {{.GoName}} {{.TypeDef}}
        n := len(valueList)
        if n != 1 {
            return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Expected one value for {{.ParamName}}, got %d", n))
        }
{{if .IsPassThrough}}
        params.{{.GoName}} = {{if not .Required}}&{{end}}valueList[0]
{{end}}
{{if .IsJson}}
        err = json.Unmarshal([]byte(valueList[0]), &{{.GoName}})
        if err != nil {
            return echo.NewHTTPError(http.StatusBadRequest, "Error unmarshaling parameter '{{.ParamName}}' as JSON")
        }
{{end}}
{{if .IsStyled}}
        err = runtime.BindStyledParameter("{{.Style}}",{{.Explode}}, "{{.ParamName}}", valueList[0], &{{.GoName}})
        if err != nil {
            return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter {{.ParamName}}: %s", err))
        }
{{end}}
        params.{{.GoName}} = {{if not .Required}}&{{end}}{{.GoName}}
        } {{if .Required}}else {
            return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Header parameter {{.ParamName}} is required, but not found"))
        }{{end}}
{{end}}
{{end}}

{{range .CookieParams}}
    if cookie, err := ctx.Cookie("{{.ParamName}}"); err == nil {
    {{if .IsPassThrough}}
    params.{{.GoName}} = {{if not .Required}}&{{end}}cookie.Value
    {{end}}
    {{if .IsJson}}
    var value {{.TypeDef}}
    var decoded string
    decoded, err := url.QueryUnescape(cookie.Value)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "Error unescaping cookie parameter '{{.ParamName}}'")
    }
    err = json.Unmarshal([]byte(decoded), &value)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "Error unmarshaling parameter '{{.ParamName}}' as JSON")
    }
    params.{{.GoName}} = {{if not .Required}}&{{end}}value
    {{end}}
    {{if .IsStyled}}
    var value {{.TypeDef}}
    err = runtime.BindStyledParameter("simple",{{.Explode}}, "{{.ParamName}}", cookie.Value, &value)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter {{.ParamName}}: %s", err))
    }
    params.{{.GoName}} = {{if not .Required}}&{{end}}value
    {{end}}
    }{{if .Required}} else {
        return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query argument {{.ParamName}} is required, but not found"))
    }{{end}}

{{end}}{{/* .CookieParams */}}

{{end}}{{/* .RequiresParamObject */}}
    // Invoke the callback with all the unmarshalled arguments
    err = w.Handler.{{.OperationId}}(ctx{{genParamNames .PathParams}}{{if .RequiresParamObject}}, params{{end}})
    return err
}
{{end}}
`,
}

// Parse parses declared templates.
func Parse(t *template.Template) (*template.Template, error) {
	for name, s := range templates {
		var tmpl *template.Template
		if t == nil {
			t = template.New(name)
		}
		if name == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(name)
		}
		if _, err := tmpl.Parse(s); err != nil {
			return nil, err
		}
	}
	return t, nil
}

