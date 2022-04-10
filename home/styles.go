package home

import (
	"fmt"
	"github.com/kib357/less-go"
	"log"
)

// CompileStylesheets Compile and minify .LESS files
func CompileStylesheets() {
	staticFolder := "./static/styles/%s"
	err := less.RenderFile(fmt.Sprintf(staticFolder, "style.less"), fmt.Sprintf(staticFolder, "style.css"), map[string]interface{}{"compress": true})
	if err != nil {
		log.Fatal(err)
	}
}
