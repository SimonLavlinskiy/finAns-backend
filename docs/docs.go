// Package docs finAns API swagger docs.
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "swagger": "2.0",
    "info": {
        "description": "REST API for finAns personal finance admin",
        "title": "finAns API",
        "version": "1.0"
    },
    "basePath": "/",
    "paths": {
        "/api/v1/health": {
            "get": {
                "description": "Returns service and database connectivity status",
                "produces": ["application/json"],
                "tags": ["health"],
                "summary": "Health check",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it.
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "finAns API",
	Description:      "REST API for finAns personal finance admin",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
