package cmd

import (
	log "github.com/Sirupsen/logrus"
	"github.com/fatih/color"
	"github.com/hokaccha/go-prettyjson"
)

func Pretty(input string, colorOutput bool) string {

	f := prettyjson.NewFormatter()
	f.Indent = 4
	if !colorOutput {
		f.DisabledColor = true
	}
	f.KeyColor = color.New(color.FgRed)
	f.NullColor = color.New(color.Underline)

	formatted, err := f.Format([]byte(input))

	if err != nil {
		log.Panic("Error formatting JSON: ", err)
	} 

	return string(formatted)

}
