package cmd

// kr8 extended logging
/*

The zerolog module used in the package forces the main program to exit if
 a Fatal or Panic event is called. With larger and more complex manifest
 generation jobs the process of error resolution is difficult when a
 holistic review of all errors during the run would be more beneficial.

The debuglog (et al) functions require an Error type input only for special
 message handling, otherwise `nil` would suffice.

This will (should) not conflict with direct use of zerolog facilities.

Usage examples

	debuglog(nil).Msg("Using component directory: " + componentDir)

	infolog(err).Str("cluster", clusterName).
				Str("component", e).
				Msg("Deleting generated for component")

	warnlog(err).Msg("Something")

	errorlog(nil).Msg("A wild error appeared!")

	fatalog(err).Msg("Error evaluating jsonnet snippet")

	fatalog(nil).Msg("Who will think of the children?!")


*/

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func debuglog(err error) *zerolog.Event {
	return log.Logger.Debug().Err(err)
}

func infolog(err error) *zerolog.Event {
	return log.Logger.Info().Err(err)
}

func warnlog(err error) *zerolog.Event {
	return log.Logger.Warn().Err(err)
}

func errorlog(err error) *zerolog.Event {
	return log.Logger.Error().Err(err)
}

func fatalog(err error) *zerolog.Event {
	if long {
		if err != nil {
			color := "\033[33m"
			nocolor := "\033[0m"
			fmt.Println(string(color), err, string(nocolor))
		}
	}
	if noexit {
		return log.WithLevel(zerolog.FatalLevel).Err(err)
	}
	return log.Fatal().Err(err) // If no conditions are met, push Fatal() event and exit the program
}

func paniclog(err error) *zerolog.Event {
	if noexit {
		return log.Logger.WithLevel(zerolog.PanicLevel)
	}
	return log.Logger.Panic().Err(err) // If no conditions are met, push Panic() event and exit the program
}
