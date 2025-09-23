package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/TylerHendrickson/csv2json/assoc"
	"github.com/TylerHendrickson/csv2json/csvmap"
	"github.com/TylerHendrickson/csv2json/internal/csvctx"
	"github.com/alecthomas/kong"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	textEncoding "golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// reportedErr is a wrapper for errors that do not need to be reported by Kong.
type reportedErr struct {
	error
}

// CLI is the command-line application root.
type CLI struct {
	CSVFile *os.File ` arg:"" help:"The CSV source file to transform. Use \"-\" to read from standard input. [default: \"${default}\"]" default:"-" name:"FILE" env:"CSV_FILE"`

	Version     kong.VersionFlag `help:"Print version information and exit."`
	VersionFull bool             `help:"Print detailed version information and exit."`
	FieldNames  []string         `name:"fields" help:"Ordered CSV column names. When set, the first row will be treated as data, not a header. [default: (determine fields from CSV header row.)]" placeholder:"NAME" env:"CSV_FIELDS"`

	OutputOpts struct {
		AsArray    bool          `name:"array" help:"Output a JSON array of parsed records. [default: (outputs newline-delimited JSON.)]" env:"ARRAY"`
		FlushEvery time.Duration `help:"How often to write results to the output. If 0, write each record immediately (no wait). Negative values disable the timer and write according to --output-buffer-size only. Output is always fully written on shutdown. [default: ${default}]" default:"1s" placeholder:"DURATION" env:"FLUSH_INTERVAL"`
		BufferSize SizeBytes     `help:"How much data to collect before writing to output. [default: (system default)]" default:"0" placeholder:"BYTES" env:"BUFFER_SIZE"`
	} `embed:"" group:"Output Options" prefix:"output-" envprefix:"OUTPUT_"`

	CSVParserOpts struct {
		FieldDelimiter   CSVDelimiter  `help:"Delimiter for CSV fields (e.g. a comma, semicolon, tab, pipe, etc.). Must be a single unicode character or the word \"tab\" for \"\\t\". [default: \"${default}\"] " placeholder:"CHAR|tab" default:"," env:"FIELD_DELIMITER"`
		CommentDelimiter *CSVDelimiter `help:"Lines starting with this character are ignored, e.g. \"#\". Must be a single unicode character or the word \"tab\" for \"\\t\". [default: (no lines will be ignored.)]" placeholder:"CHAR|tab" optional:"" env:"COMMENT_DELIMITER"`
		LazyQuotes       bool          `help:"A quote may appear in an unquoted field and a non-doubled quote may appear in a quoted field. " env:"LAZY_QUOTES"`
		TrimLeadingSpace bool          `help:"Leading white space in a field is ignored (even if --csv-field-delimiter is a white space character). " env:"TRIM_LEADING_SPACE"`
	} `embed:"" prefix:"csv-" group:"CSV Parser Options" envprefix:"CSV_PARSER_"`

	ErrorHandlingOpts struct {
		OnParseError  OnErrorAction `help:"Action the program should take for a record that fails to parse. [default: ${default}] " default:"abort" enum:"${onParseErrorEnum}" env:"ON_PARSE_ERROR"`
		OnValuesError OnErrorAction `help:"Action the program should take for a record that has an unexpected number of values. NOTE: If \"allow\", missing fields become empty strings and additional fields are dropped. [default: ${default}] " default:"abort" enum:"${onValuesErrorEnum}" env:"ON_VALUES_ERROR"`
	} `embed:"" group:"Error-Handling Behaviors" help:"Behaviors for handling errors"`

	LoggingOpts struct {
		Level  zerolog.Level `help:"Minimum log level. [default: ${default}] " enum:"${logLevelEnum}" default:"warn" env:"LOG_LEVEL"`
		Format struct {
			Pretty bool `help:"Force pretty log output. [default: (enabled if stderr is a TTY.)] " xor:"logfmt" env:"LOG_PRETTY"`
			JSON   bool `help:"Force JSON log output. [default: (enabled if stderr is not a TTY.)]" xor:"logfmt" env:"LOG_JSON"`
		} `embed:""`
		TimestampLayout string `help:"Layout for formatting logged timestamps. Expects a Go time layout string. [default: \"${default}\" (${logTimestampDefaultName})] " default:"${logTimestampDefaultLayout}" placeholder:"LAYOUT" env:"LOG_TIMESTAMP_LAYOUT"`
		IncludeRecords  bool   `help:"Include transformed records in log output. " name:"records" env:"LOG_INCLUDE_RECORDS"`
		NoColor         bool   `help:"Disable colorized log output (affects pretty logs only). " default:"false" env:"NO_COLOR,LOG_NO_COLOR"`
	} `embed:"" prefix:"log-" group:"Logging Options" description:"Control Logging Behaviors"`
}

// newLogger creates and returns a new logger according to the CLI configuration state.
func (cli *CLI) newLogger() zerolog.Logger {
	zerolog.TimeFieldFormat = cli.LoggingOpts.TimestampLayout
	var logWriter io.Writer = os.Stderr
	if (isatty.IsTerminal(os.Stderr.Fd()) || cli.LoggingOpts.Format.Pretty) && !cli.LoggingOpts.Format.JSON {
		logWriter = zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.Out = logWriter
			w.TimeFormat = cli.LoggingOpts.TimestampLayout
			w.NoColor = cli.LoggingOpts.NoColor
		})
	}
	logger := zerolog.New(logWriter).With().
		Timestamp().
		Logger().
		Level(zerolog.Level(cli.LoggingOpts.Level))
	if logger.GetLevel() == zerolog.TraceLevel {
		// Add caller to all logs when minimum log level is trace
		logger = logger.With().Caller().Logger()
	}
	return logger
}

// newCSVReader creates and returns a new CSV reader according to the CLI configuration state.
func (cli *CLI) newCSVReader() *csv.Reader {
	reader := csv.NewReader(
		// Transformer drops BOM from the start of the file if one is present,
		// then falls back to a no-op transformer.
		// This is useful because BOM presence interferes with "encodings/csv" quote-handling.
		transform.NewReader(cli.CSVFile, unicode.BOMOverride(textEncoding.Nop.NewDecoder())),
	)
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = false
	reader.Comma = rune(cli.CSVParserOpts.FieldDelimiter)
	if cli.CSVParserOpts.CommentDelimiter != nil {
		reader.Comment = rune(*cli.CSVParserOpts.CommentDelimiter)
	}
	reader.LazyQuotes = cli.CSVParserOpts.LazyQuotes
	reader.TrimLeadingSpace = cli.CSVParserOpts.TrimLeadingSpace
	return reader
}

// newCSVContextReader wraps a new *csv.Reader configured by CLI.newCSVReader()
// in a *csvContextReader for context-aware reads.
func (cli *CLI) newCSVContextReader(ctx context.Context) *csvctx.Reader {
	return csvctx.New(ctx, cli.newCSVReader())
}

// newRecordWriter creates a new transformation result writer according to CLI configuration
func (cli *CLI) newRecordWriter() *jsonRecordWriter {
	return NewJSONRecordWriter(os.Stdout, cli.OutputOpts.AsArray, int(cli.OutputOpts.BufferSize))
}

// Validate is a hook that performs additional validation of parsed CLI configuration.
func (cli *CLI) Validate() error {
	if cli.CSVParserOpts.CommentDelimiter != nil {
		if val := cli.CSVParserOpts.FieldDelimiter; val == *cli.CSVParserOpts.CommentDelimiter {
			return fmt.Errorf("%s cannot have the same value (%q) as %s",
				"--csv-comment-delimiter", val, "--csv-field-delimiter")
		}
	}
	return nil
}

// AfterApply is a hook that configures the application after parsing.
func (cli *CLI) AfterApply(ctx context.Context, kctx *kong.Context) error {
	logger := cli.newLogger().With().Str("input", cli.CSVFile.Name()).Logger()
	reader := cli.newCSVContextReader(ctx)
	writer := cli.newRecordWriter()

	logger.Trace().Interface("configuration", cli).Msg("dump final application configuration")
	kctx.Bind(logger, reader, writer)
	logger.Debug().
		// zerolog.Array.Type() does not exist; see https://github.com/rs/zerolog/issues/729
		// Array("bound-types", zerolog.Arr().Type(logger).Type(reader).Type(writer)).
		Array("bound-types", zerolog.Arr().
			Str(fmt.Sprintf("%T", logger)).
			Str(fmt.Sprintf("%T", reader)).
			Str(fmt.Sprintf("%T", writer)),
		).
		Msg("adding bindings to application context")
	return nil
}

// Run is the primary hook that runs the CLI application.
// It reads records from r until EOF and transforms each record into JSON output written to w.
// Returns an error when further execution is cannot continue due to requested shutdown
// or when CSV parsing results in an error for which the application is configured to abort.
func (cli *CLI) Run(ctx context.Context, logger zerolog.Logger, r *csvctx.Reader, w *jsonRecordWriter) (err error) {
	defer func() {
		logger.Debug().Msg("closing input stream")
		if cerr := cli.CSVFile.Close(); cerr != nil {
			logger.Err(err).Msg("error closing input stream")
		}

		logger.Debug().Msg("closing output stream")
		if cerr := w.Close(); cerr != nil {
			logger.Err(cerr).Msg("error closing output stream")
		}
	}()

	logger.Debug().Msg("determining field names for mapper")
	fieldNames, err := cli.getFieldNames(logger, r)
	if err != nil {
		logger := logger.With().Err(err).Logger()
		if csvctx.IsContextError(err) {
			logger.Warn().Msg("shutdown requested while getting CSV field names")
		} else {
			logger.Error().Msg("error getting CSV field names")
		}
		return reportedErr{err}
	}
	mapper := csvmap.NewFromRowSource(r, fieldNames)

	// Begin flushing the output buffer periodically according to CLI configuration
	stopPeriodicFlush := startPeriodicFlush(w, cli.OutputOpts.FlushEvery)
	defer stopPeriodicFlush()

	logger.Info().Msg("ready to receive CSV data")
	for {
		logger.Debug().Msg("waiting to read next CSV record")
		result := mapper.Next()
		err = result.Err

		// Abort immediately if context was cancelled
		if csvctx.IsContextError(err) {
			logger.Warn().Err(err).Msg("shutdown requested while reading next CSV record")
			return reportedErr{err}
		}

		if err == io.EOF {
			logger.Debug().Msg("encountered EOF")
			err = nil
			break
		}

		lineLogger := logger.With().Int("csv-line", result.Line).Logger()
		if cli.LoggingOpts.IncludeRecords {
			lineLogger = lineLogger.With().Any("record", result.Record).Logger()
		}
		action := allow

		if err != nil {
			lineLogger = lineLogger.With().Err(err).Logger()
			switch err {
			case assoc.ErrValuesFewerThanKeys:
				explanation := "row does not contain enough field values to map all columns"
				lineLogger = lineLogger.With().
					Str("error-type", "values").
					Str("explanation", explanation).
					Logger()
			case assoc.ErrValuesExceedKeys:
				action = cli.ErrorHandlingOpts.OnValuesError
				lineLogger = lineLogger.With().
					Str("error-type", "values").
					Str("explanation", "row has more fields than header names; extra fields will be ignored").
					Logger()
			default:
				lineLogger = lineLogger.With().
					Str("error-type", "parse").
					Str("explanation", "error reading file or invalid CSV content").
					Logger()
				action = cli.ErrorHandlingOpts.OnParseError
			}
			lineLogger = lineLogger.With().Stringer("action", action).Logger()
			lineLogger.Warn().Msg("error encountered when reading CSV record")
		}

		switch action {
		case allow:
			lineLogger.Info().Msg("writing transformed record to output stream")
			if _, err := w.WriteRecord(result.Record); err != nil {
				lineLogger.Err(err).Msg("error writing transformed record to output stream")
			}
		case null:
			lineLogger.Info().Msg("writing null record to output stream")
			if _, err := w.WriteNull(); err != nil {
				lineLogger.Err(err).Msg("error writing null record to output stream")
			}
		case skip:
			lineLogger.Info().Msg("skipping output for record")
		case abort:
			lineLogger.Error().Msg("aborting execution due to error")
			return reportedErr{err}
		}

		if cli.OutputOpts.FlushEvery == 0 {
			lineLogger.Debug().Msg("flushing output buffer immediately")
			if flushErr := w.Flush(); flushErr != nil {
				lineLogger.Error().Err(flushErr).Msg("error flushing output buffer")
			}
		}
	}

	logger.Info().Msg("no more records")
	return nil
}

// getFieldNames returns fieldnames configured on cli, if any, or else reads the
// first row of data to get fieldnames from the CSV header.
// This means that when cli.FieldNames has values, the first row of the CSV will
// be treated as a data row, not a header row.
// Returns errors from r.Read().
func (cli *CLI) getFieldNames(logger zerolog.Logger, r interface{ Read() ([]string, error) }) ([]string, error) {
	if len(cli.FieldNames) > 0 {
		return cli.FieldNames, nil
	}

	logger.Debug().Msg("reading CSV header row to determine field names")
	cancelWaitLogging := DoAfterUnlessCanceled(5*time.Second, func() {
		logger.Info().Msg("still waiting for CSV header row input")
	})

	fieldNames, err := r.Read()
	cancelWaitLogging()
	if err != nil {
		return nil, err
	}

	logger.Debug().
		Int("field-count", len(fieldNames)).
		Msg("derived field names from CSV header row")

	return fieldNames, nil
}
