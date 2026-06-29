package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/interviewos/backend/api"
)

// swaggerHTML is a self-contained Swagger UI page. The UI bundle is loaded from
// the unpkg CDN (allowed by the Content-Security-Policy) and pointed at the
// embedded spec served from /swagger/openapi.yaml on the same origin.
const swaggerHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>InterviewOS API — Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <style>
    html, body { margin: 0; padding: 0; background: #fafafa; }
    #swagger-ui { max-width: 1460px; margin: 0 auto; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js" crossorigin></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({
        url: "openapi.yaml",
        dom_id: "#swagger-ui",
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        layout: "StandaloneLayout"
      });
    };
  </script>
</body>
</html>`

// RegisterSwagger mounts the interactive API docs: a Swagger UI page at
// /swagger (and /swagger/) plus the embedded OpenAPI spec at
// /swagger/openapi.yaml. These are unauthenticated and intentionally cheap.
func RegisterSwagger(engine *gin.Engine) {
	serveUI := func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
	}
	engine.GET("/swagger", serveUI)
	engine.GET("/swagger/", serveUI)

	engine.GET("/swagger/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", api.OpenAPISpec)
	})
}
