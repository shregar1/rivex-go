package rivex

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type OpenAPIRequest struct {
	Body        interface{}
	Required    bool
	Description string
	ContentType string
}

type OpenAPIResponse struct {
	Body        interface{}
	Description string
}

type OpenAPIOperation struct {
	Summary     string
	Description string
	Tags        []string
	Parameters  []OpenAPIParam
	RequestBody *OpenAPIRequest
	Responses   map[string]*OpenAPIResponse
	Deprecated  bool
	Security    []map[string][]string
}

type OpenAPIParam struct {
	Name        string
	In          string
	Description string
	Required    bool
	Type        string
	Format      string
	Schema      interface{}
	Example     interface{}
}

type OpenAPIRouteMetadata struct {
	Path    string
	Method  string
	Operation *OpenAPIOperation
}

func GenerateOpenAPI(app *Rivex) map[string]interface{} {
	spec := map[string]interface{}{
		"openapi": "3.1.0",
		"info": map[string]interface{}{
			"title":   app.title,
			"version": app.version,
		},
	}
	if app.description != "" {
		spec["info"].(map[string]interface{})["description"] = app.description
	}
	if app.servers != nil {
		spec["servers"] = app.servers
	}
	paths := make(map[string]interface{})
	components := make(map[string]interface{})
	allTags := make(map[string]struct{})
	for _, route := range app.routes {
		if !route.IncludeInSchema {
			continue
		}
		pathKey := route.Path
		methodKey := strings.ToLower(route.Method)
		pathMethods, ok := paths[pathKey]
		if !ok {
			pathMethods = make(map[string]interface{})
			paths[pathKey] = pathMethods
		}
		op := buildOperation(route, components)
		pathMethods.(map[string]interface{})[methodKey] = op
		for _, tag := range route.Tags {
			allTags[tag] = struct{}{}
		}
	}
	if len(paths) > 0 {
		spec["paths"] = paths
	}
	if len(components) > 0 {
		spec["components"] = components
	}
	if app.openAPITags != nil {
		spec["tags"] = app.openAPITags
	} else if len(allTags) > 0 {
		tagNames := make([]string, 0, len(allTags))
		for t := range allTags {
			tagNames = append(tagNames, t)
		}
		sort.Strings(tagNames)
		tags := make([]map[string]interface{}, len(tagNames))
		for i, name := range tagNames {
			tags[i] = map[string]interface{}{"name": name}
		}
		spec["tags"] = tags
	}
	return spec
}

func buildOperation(route RouteRecord, components map[string]interface{}) map[string]interface{} {
	op := make(map[string]interface{})
	if route.Summary != "" {
		op["summary"] = route.Summary
	}
	if route.Description != "" {
		op["description"] = route.Description
	}
	op["operationId"] = generateOperationID(route)
	if len(route.Tags) > 0 {
		op["tags"] = route.Tags
	}
	if route.Deprecated {
		op["deprecated"] = true
	}
	params := extractOpenAPIParams(route.Path)
	for _, p := range params {
		op["parameters"] = appendToParameters(op, p)
	}
	responses := map[string]interface{}{
		"200": map[string]interface{}{"description": "Successful response"},
	}
	if route.StatusCode != 0 && route.StatusCode != 200 {
		statusKey := fmt.Sprintf("%d", route.StatusCode)
		responses[statusKey] = map[string]interface{}{"description": StatusText(route.StatusCode)}
	}
	responses["422"] = map[string]interface{}{"description": "Validation error"}
	responses["500"] = map[string]interface{}{"description": "Server error"}
	op["responses"] = responses
	return op
}

func generateOperationID(route RouteRecord) string {
	if route.OperationID != "" {
		return route.OperationID
	}
	pathSlug := strings.Trim(route.Path, "/")
	pathSlug = strings.ReplaceAll(pathSlug, "/", "_")
	pathSlug = strings.ReplaceAll(pathSlug, "{", "")
	pathSlug = strings.ReplaceAll(pathSlug, "}", "")
	if pathSlug == "" {
		return strings.ToLower(route.Method) + "_root"
	}
	return strings.ToLower(route.Method) + "_" + pathSlug
}

func extractOpenAPIParams(path string) []OpenAPIParam {
	var params []OpenAPIParam
	for _, segment := range strings.Split(path, "/") {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			name := segment[1 : len(segment)-1]
			params = append(params, OpenAPIParam{
				Name:     name,
				In:       "path",
				Required: true,
				Type:     "string",
				Schema:   map[string]interface{}{"type": "string"},
			})
		}
	}
	return params
}

func appendToParameters(op map[string]interface{}, p OpenAPIParam) []map[string]interface{} {
	var params []map[string]interface{}
	if existing, ok := op["parameters"].([]map[string]interface{}); ok {
		params = existing
	}
	paramMap := map[string]interface{}{
		"name": p.Name,
		"in":   p.In,
	}
	if p.Description != "" {
		paramMap["description"] = p.Description
	}
	if p.Required {
		paramMap["required"] = true
	}
	if p.Schema != nil {
		paramMap["schema"] = p.Schema
	} else if p.Type != "" {
		schema := map[string]interface{}{"type": p.Type}
		if p.Format != "" {
			schema["format"] = p.Format
		}
		paramMap["schema"] = schema
	}
	if p.Example != nil {
		paramMap["example"] = p.Example
	}
	params = append(params, paramMap)
	return params
}

func GenerateSchemaFromType(t reflect.Type, components map[string]interface{}) map[string]interface{} {
	return generateSchemaFromType(t, components, make(map[reflect.Type]bool))
}

func generateSchemaFromType(t reflect.Type, components map[string]interface{}, visited map[reflect.Type]bool) map[string]interface{} {
	if t == nil {
		return map[string]interface{}{"type": "object"}
	}
	if visited[t] {
		return map[string]interface{}{"$ref": "#/components/schemas/" + t.Name()}
	}
	switch t.Kind() {
	case reflect.String:
		return map[string]interface{}{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]interface{}{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number"}
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.Slice, reflect.Array:
		return map[string]interface{}{
			"type":  "array",
			"items": generateSchemaFromType(t.Elem(), components, visited),
		}
	case reflect.Map:
		return map[string]interface{}{
			"type":                 "object",
			"additionalProperties": generateSchemaFromType(t.Elem(), components, visited),
		}
	case reflect.Ptr:
		return generateSchemaFromType(t.Elem(), components, visited)
	case reflect.Struct:
		visited[t] = true
		properties := make(map[string]interface{})
		var required []string
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			name := strings.ToLower(field.Name)
			jsonTag := field.Tag.Get("json")
			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] == "-" {
					continue
				}
				if parts[0] != "" {
					name = parts[0]
				}
				for _, opt := range parts[1:] {
					if opt == "omitempty" {
						continue
					}
				}
			}
			properties[name] = generateSchemaFromType(field.Type, components, visited)
			if validateTag := field.Tag.Get("validate"); strings.Contains(validateTag, "required") {
				required = append(required, name)
			}
		}
		schema := map[string]interface{}{
			"type":       "object",
			"properties": properties,
		}
		if len(required) > 0 {
			schema["required"] = required
		}
		if t.Name() != "" {
			if components == nil {
				components = make(map[string]interface{})
			}
			schemas, ok := components["schemas"].(map[string]interface{})
			if !ok {
				schemas = make(map[string]interface{})
				components["schemas"] = schemas
			}
			schemas[t.Name()] = schema
			return map[string]interface{}{"$ref": "#/components/schemas/" + t.Name()}
		}
		return schema
	}
	return map[string]interface{}{"type": "object"}
}

func GenerateOpenAPIRequestBody(body interface{}, required bool, components map[string]interface{}) *OpenAPIRequest {
	return &OpenAPIRequest{
		Body:        GenerateSchemaFromType(reflect.TypeOf(body), components),
		Required:    required,
		ContentType: "application/json",
	}
}

func GenerateOpenAPIResponse(body interface{}, description string, components map[string]interface{}) *OpenAPIResponse {
	return &OpenAPIResponse{
		Body:        GenerateSchemaFromType(reflect.TypeOf(body), components),
		Description: description,
	}
}

func BuildOperationWithRequestResponse(method, path string, op *OpenAPIOperation, components map[string]interface{}) map[string]interface{} {
	operation := map[string]interface{}{}
	if op.Summary != "" {
		operation["summary"] = op.Summary
	}
	if op.Description != "" {
		operation["description"] = op.Description
	}
	if len(op.Tags) > 0 {
		operation["tags"] = op.Tags
	}
	if op.Deprecated {
		operation["deprecated"] = true
	}
	operation["operationId"] = generateOperationIDFor(method, path)
	params := extractOpenAPIParams(path)
	for _, p := range params {
		operation["parameters"] = appendToParameters(operation, p)
	}
	if op.RequestBody != nil {
		schema := map[string]interface{}{}
		if op.RequestBody.Body != nil {
			schema = GenerateSchemaFromType(reflect.TypeOf(op.RequestBody.Body), components)
		}
		operation["requestBody"] = map[string]interface{}{
			"required": op.RequestBody.Required,
			"content": map[string]interface{}{
				op.RequestBody.ContentType: map[string]interface{}{
					"schema": schema,
				},
			},
		}
		if op.RequestBody.Description != "" {
			operation["requestBody"].(map[string]interface{})["description"] = op.RequestBody.Description
		}
	}
	if len(op.Responses) > 0 {
		responses := map[string]interface{}{}
		for status, resp := range op.Responses {
			content := map[string]interface{}{}
			if resp != nil && resp.Body != nil {
				schema := GenerateSchemaFromType(reflect.TypeOf(resp.Body), components)
				content["application/json"] = map[string]interface{}{"schema": schema}
			}
			r := map[string]interface{}{}
			if resp != nil && resp.Description != "" {
				r["description"] = resp.Description
			} else {
				r["description"] = StatusText(atoi(status))
			}
			if len(content) > 0 {
				r["content"] = content
			}
			responses[status] = r
		}
		operation["responses"] = responses
	} else {
		operation["responses"] = map[string]interface{}{
			"200": map[string]interface{}{"description": "Successful response"},
		}
	}
	if len(op.Security) > 0 {
		operation["security"] = op.Security
	}
	return operation
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return 0
		}
	}
	return n
}

func generateOperationIDFor(method, path string) string {
	pathSlug := strings.Trim(path, "/")
	pathSlug = strings.ReplaceAll(pathSlug, "/", "_")
	pathSlug = strings.ReplaceAll(pathSlug, "{", "")
	pathSlug = strings.ReplaceAll(pathSlug, "}", "")
	if pathSlug == "" {
		return strings.ToLower(method) + "_root"
	}
	return strings.ToLower(method) + "_" + pathSlug
}

func BuildOpenAPIPath(method, path string, op *OpenAPIOperation) map[string]interface{} {
	return map[string]interface{}{
		strings.ToLower(method): BuildOperationWithRequestResponse(method, path, op, nil),
	}
}

func RegisterSchemaComponent(components map[string]interface{}, name string, schema interface{}) {
	if components == nil {
		return
	}
	schemas, ok := components["schemas"].(map[string]interface{})
	if !ok {
		schemas = make(map[string]interface{})
		components["schemas"] = schemas
	}
	schemas[name] = schema
}

func AppendOpenAPIPath(spec map[string]interface{}, method, path string, operation map[string]interface{}) {
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		paths = make(map[string]interface{})
		spec["paths"] = paths
	}
	pathMethods, ok := paths[path].(map[string]interface{})
	if !ok {
		pathMethods = make(map[string]interface{})
		paths[path] = pathMethods
	}
	pathMethods[strings.ToLower(method)] = operation
}

func generateDocsHTML(title, openAPIURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <title>%s - API Docs</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        :root {
            --bg: #fff;
            --fg: #050505;
            --muted: #676767;
            --line: #dedede;
            --soft: #f6f6f6;
            --mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
            --sans: Inter, ui-sans-serif, system-ui, -apple-system, sans-serif;
        }
        * { box-sizing: border-box; }
        body {
            margin: 0; background: var(--bg); color: var(--fg);
            font-family: var(--sans);
        }
        .shell { min-height: 100vh; padding: clamp(20px, 3vw, 44px); }
        .topbar {
            display: flex; justify-content: space-between; gap: 22px;
            border-bottom: 1px solid var(--line); padding-bottom: 18px;
        }
        .brand {
            font-family: var(--mono); font-size: 12px; font-weight: 800;
            letter-spacing: .12em; text-transform: uppercase;
        }
        h1 {
            margin: 40px 0 20px; font-size: clamp(36px, 8vw, 96px);
            line-height: .9; text-transform: uppercase;
        }
        .endpoint {
            border-bottom: 1px solid var(--line); padding: 18px 0;
        }
        .method {
            display: inline-block; border: 1px solid var(--fg);
            background: var(--fg); color: var(--bg);
            font-family: var(--mono); font-size: 12px; font-weight: 800;
            text-align: center; padding: 6px 12px; min-width: 70px;
        }
        .path {
            font-family: var(--mono); font-size: 18px; font-weight: 700;
            margin-left: 16px;
        }
        .empty {
            border: 1px solid var(--line); background: var(--soft);
            color: var(--muted); padding: 24px; margin-top: 20px;
        }
    </style>
</head>
<body>
    <main class="shell">
        <header class="topbar">
            <div class="brand">RIVEX API DOCS</div>
        </header>
        <h1>%s</h1>
        <p id="description">Loading OpenAPI specification...</p>
        <div id="endpoint-list"><div class="empty">Loading endpoints...</div></div>
    </main>
    <script>
        fetch("%s").then(r => r.json()).then(spec => {
            const desc = document.getElementById("description");
            desc.textContent = spec.info?.description || "API documentation";
            const list = document.getElementById("endpoint-list");
            const paths = spec.paths || {};
            const methods = ["get","post","put","patch","delete","head","options"];
            let html = "";
            for (const [path, ops] of Object.entries(paths)) {
                for (const m of methods) {
                    if (!ops[m]) continue;
                    const op = ops[m];
                    html += '<div class="endpoint">' +
                        '<span class="method">' + m.toUpperCase() + '</span>' +
                        '<span class="path">' + path + '</span>' +
                        (op.summary ? '<p style="color:var(--muted);margin:8px 0 0 88px">' + op.summary + '</p>' : '') +
                        '</div>';
                }
            }
            list.innerHTML = html || '<div class="empty">No endpoints found.</div>';
        }).catch(err => {
            document.getElementById("endpoint-list").innerHTML =
                '<div class="empty">Failed to load spec: ' + err.message + '</div>';
        });
    </script>
</body>
</html>`, title, title, openAPIURL)
}
