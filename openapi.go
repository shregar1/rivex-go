package rivex

import (
	"fmt"
	"sort"
	"strings"
)

// GenerateOpenAPI generates an OpenAPI 3.1 specification from the application's
// registered routes.
func GenerateOpenAPI(app *Rivex) map[string]interface{} {
	spec := map[string]interface{}{
		"openapi": "3.1.0",
		"info": map[string]interface{}{
			"title":   app.title,
			"version": app.version,
		},
	}

	if app.description != "" {
		info := spec["info"].(map[string]interface{})
		info["description"] = app.description
	}

	if app.servers != nil {
		spec["servers"] = app.servers
	}

	paths := make(map[string]interface{})
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

		operation := buildOperation(route)
		pathMethods.(map[string]interface{})[methodKey] = operation

		for _, tag := range route.Tags {
			allTags[tag] = struct{}{}
		}
	}

	if len(paths) > 0 {
		spec["paths"] = paths
	}

	if app.openAPITags != nil {
		spec["tags"] = app.openAPITags
	} else if len(allTags) > 0 {
		var tagNames []string
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

// buildOperation creates an OpenAPI operation object from a route record.
func buildOperation(route RouteRecord) map[string]interface{} {
	op := make(map[string]interface{})

	if route.Summary != "" {
		op["summary"] = route.Summary
	}

	if route.Description != "" {
		op["description"] = route.Description
	}

	if route.OperationID != "" {
		op["operationId"] = route.OperationID
	} else {
		// Generate an operation ID from the method and path
		pathSlug := strings.Trim(route.Path, "/")
		pathSlug = strings.ReplaceAll(pathSlug, "/", "_")
		pathSlug = strings.ReplaceAll(pathSlug, "{", "")
		pathSlug = strings.ReplaceAll(pathSlug, "}", "")
		if pathSlug == "" {
			op["operationId"] = strings.ToLower(route.Method) + "_root"
		} else {
			op["operationId"] = strings.ToLower(route.Method) + "_" + pathSlug
		}
	}

	if len(route.Tags) > 0 {
		op["tags"] = route.Tags
	}

	if route.Deprecated {
		op["deprecated"] = true
	}

	// Extract path parameters from the pattern
	params := extractPathParams(route.Path)
	if len(params) > 0 {
		parameters := make([]map[string]interface{}, len(params))
		for i, name := range params {
			parameters[i] = map[string]interface{}{
				"name":     name,
				"in":       "path",
				"required": true,
				"schema":   map[string]interface{}{"type": "string"},
			}
		}
		op["parameters"] = parameters
	}

	// Default response
	statusKey := fmt.Sprintf("%d", route.StatusCode)
	op["responses"] = map[string]interface{}{
		statusKey: map[string]interface{}{
			"description": "Successful response",
		},
	}

	return op
}

// extractPathParams extracts parameter names from a path pattern like
// "/users/{id}/posts/{post_id}".
func extractPathParams(path string) []string {
	var params []string
	for _, segment := range strings.Split(path, "/") {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			params = append(params, segment[1:len(segment)-1])
		}
	}
	return params
}

// generateDocsHTML creates the Swagger UI HTML page for the API documentation.
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
