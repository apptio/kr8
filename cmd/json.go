package cmd

import (
	"github.com/fatih/color"
	"github.com/hokaccha/go-prettyjson"
	"github.com/rs/zerolog/log"
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
		log.Fatal().Err(err).Msg("Error formatting JSON")
	}

	return string(formatted)

}
