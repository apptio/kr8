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

	paniclog(nil).Msg("Who will think of the children?!")

	//TODO: Summary report at end of run (requires some tracking facility, a run manifest)
	//TODO: Ability to dump full cluster parameters or other capabilities
	//TODO: Not logging, but some validation. Should that be in kr8?
	//TODO: Environment Variable and config file examples?

*/

import (
    //"fmt"
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

var longcolor string = "\033[1m\033[33m" /* Bold Yellow */
var longnocolor string = "\033[0m"       // color reset

// RESET   "\033[0m"
// BLACK   "\033[30m"      /* Black */
// RED     "\033[31m"      /* Red */
// GREEN   "\033[32m"      /* Green */
// YELLOW  "\033[33m"      /* Yellow */
// BLUE    "\033[34m"      /* Blue */
// MAGENTA "\033[35m"      /* Magenta */
// CYAN    "\033[36m"      /* Cyan */
// WHITE   "\033[37m"      /* White */
// BOLDBLACK   "\033[1m\033[30m"      /* Bold Black */
// BOLDRED     "\033[1m\033[31m"      /* Bold Red */
// BOLDGREEN   "\033[1m\033[32m"      /* Bold Green */
// BOLDYELLOW  "\033[1m\033[33m"      /* Bold Yellow */
// BOLDBLUE    "\033[1m\033[34m"      /* Bold Blue */
// BOLDMAGENTA "\033[1m\033[35m"      /* Bold Magenta */
// BOLDCYAN    "\033[1m\033[36m"      /* Bold Cyan */
// BOLDWHITE   "\033[1m\033[37m"      /* Bold White */

func tracelog(err error) *zerolog.Event {
    return log.Logger.Trace().Err(err)
}

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
    updateExitCode(1)
    return log.Logger.Error().Err(err)
}

func fatalog(err error) *zerolog.Event {

    if noexit {
        updateExitCode(2)
        return log.WithLevel(zerolog.FatalLevel).Err(err)
    }
    return log.Fatal().Err(err) // If no conditions are met, push Fatal() event and exit the program
}

func paniclog(err error) *zerolog.Event {
    if noexit {
        updateExitCode(3)
        return log.Logger.WithLevel(zerolog.PanicLevel)
    }
    return log.Logger.Panic().Err(err) // If no conditions are met, push Panic() event and exit the program
}

func updateExitCode(exitcode int) {
    if exitcode > exit {
        exit = exitcode
    }
}
