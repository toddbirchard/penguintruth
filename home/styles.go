package home

import (
	"fmt"
	"log"

	"github.com/kib357/less-go"
)

// CompileStylesheets Compile and minify .LESS files
func CompileStylesheets() {
	staticFolder := "./static/%s"
	err := less.RenderFile(fmt.Sprintf(staticFolder, "src/less/style.less"), fmt.Sprintf(staticFolder, "dist/css/style.css"), map[string]interface{}{"compress": true})
	if err != nil {
		log.Fatal(err)
	}
}
