package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
)

func main() {
	// Register context to allow graceful shutdown on SIGINT.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	// If signaled, unregister to restore default behavior and allow any
	// subsequent SIGINT to exit immediately.
	go func() { <-ctx.Done(); stop() }()
	defer stop()

	var cli CLI
	kctx := kong.Parse(
		&cli,
		kong.Description("Restructures CSV into JSON."),
		kong.BindTo(ctx, (*context.Context)(nil)),
		RegisterEnumPlaceholderMapperOpt[zerolog.Level](),
		RegisterEnumPlaceholderMapperOpt[OnErrorAction](),
		kong.Vars{
			"version":                   versionStringShort(),
			"defaultLogLevelName":       zerolog.WarnLevel.String(),
			"logTimestampDefaultName":   "RFC3339",
			"logTimestampDefaultLayout": time.RFC3339,
			"onParseErrorEnum":          enumTag(abort, null, skip),
			"onValuesErrorEnum":         enumTag(abort, allow, null, skip),
			"logLevelEnum": enumTag(
				zerolog.TraceLevel,
				zerolog.DebugLevel,
				zerolog.InfoLevel,
				zerolog.WarnLevel,
				zerolog.ErrorLevel,
				zerolog.FatalLevel,
				zerolog.PanicLevel,
			),
			"logTimestampLayoutExamples": joinQuoted(", ", time.RFC822, time.UnixDate, time.Stamp),
		},
	)

	if cli.VersionFull {
		// Print detailed version and exit
		fmt.Fprintln(kctx.Stdout, versionStringFull())
		kctx.Exit(0)
	}

	if err := kctx.Run(); err != nil {
		var re reportedErr
		if errors.As(err, &re) {
			kctx.Exit(1)
		}
		kctx.FatalIfErrorf(err)
	}
}
