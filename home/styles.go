package home

import (
	"fmt"
	"log"

	"github.com/bep/godartsass/v2"
)

type Args struct {
	// The input source.
	Source string

	// Defaults is SCSS.
	SourceSyntax SourceSyntax

	// Default is EXPANDED.
	OutputStyle EXPANDED

	// Custom resolver to use to resolve imports.
	// If set, this will be the first in the resolver chain.
	ImportResolver ImportResolver

}

type Import struct {
	// The content of the imported file.
	Content string

	// The syntax of the imported file.
	SourceSyntax SourceSyntax
}

type LogEvent struct {
	// Type is the type of log event.
	Type LogEventType

	// Message on the form url:line:col message.
	Message string
}

// CompileStylesheets Compile and minify .LESS files
func Start(opts Options) (*Transpiler, error) {
		staticFolder := "./static/%s"
	err := less.RenderFile(fmt.Sprintf(staticFolder, "src/less/style.less"), fmt.Sprintf(staticFolder, "dist/css/style.css"), map[string]interface{}{"compress": true})
	if err != nil {
		log.Fatal(err)
	}
}
