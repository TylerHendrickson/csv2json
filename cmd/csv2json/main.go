package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
)

const (
	AppVersion string = "0.2.0" // TODO: Make this dynamic?
)

func main() {
	var cli CLI
	// TODO: Handle OS signals to enable logging on SIGINT, etc.
	ctx := context.Background()
	kctx := kong.Parse(&cli,
		kong.Description("Restructures CSV into JSON."),
		kong.Bind(ctx),
		kong.Vars{
			"version":             AppVersion,
			"defaultLogLevelName": zerolog.WarnLevel.String(),
			"logLevelEnum": joinStringers(",",
				zerolog.TraceLevel,
				zerolog.DebugLevel,
				zerolog.InfoLevel,
				zerolog.WarnLevel,
				zerolog.ErrorLevel,
				zerolog.FatalLevel,
				zerolog.PanicLevel,
			),
			"onParseErrorEnum":          joinStringers(",", abort, null, skip),
			"onValuesErrorEnum":         joinStringers(",", abort, allow, null, skip),
			"logTimestampDefaultName":   "RFC3339",
			"logTimestampDefaultLayout": time.RFC3339,
		},
	)

	if err := kctx.Run(); err != nil {
		var re runErr
		if errors.As(err, &re) {
			os.Exit(1)
		}
		kctx.FatalIfErrorf(err)
	}
}
