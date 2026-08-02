package home

import (
	"fmt"

	"github.com/kib357/less-go"
	log "github.com/sirupsen/logrus"
)

// CompileStylesheets Compile and minify .LESS files
func CompileStylesheets() {
	staticFolder := "./static/%s"
	err := less.RenderFile(fmt.Sprintf(staticFolder, "src/less/style.less"), fmt.Sprintf(staticFolder, "dist/css/style.css"), map[string]interface{}{"compress": true})
	if err != nil {
		log.Fatal(err)
	}
}
