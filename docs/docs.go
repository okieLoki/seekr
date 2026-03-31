package docs

import (
	_ "embed"
	"net/http"
)

//go:embed swagger.yaml
var SwaggerYAML []byte

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(SwaggerYAML)
}
