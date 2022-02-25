package cmd

import (
	"fmt"
	"github.com/TylerHendrickson/csv2json/pkg/converter"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
	"os"
)

var (
	logger   = log.NewNopLogger()
	logLevel int
	logJson  = false
	options  converter.Options
)

var rootCmd = &cobra.Command{
	Use:     "csv2json [file]",
	Short:   "Converts CSV input to JSON output",
	Long: "joiwefijoei",
	Version: "0.4.0",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			options.CsvInput = os.Stdin
		} else {
			if f, err := os.Open(args[0]); err != nil {
				return err
			} else {
				options.CsvInput = f
			}
		}
		return converter.Execute(options)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(setUpLogs)
	options.JsonOutput = os.Stdout
	options.Logger = &logger
	rootCmd.PersistentFlags().CountVarP(
		&logLevel, "verbosity", "v", "Verbosity level for logging (default is error-only)")
	rootCmd.Flags().StringSliceVarP(&options.ColNames, "force-columns", "c", []string{},
		"Column names, which must equal the number of CSV fields if given. "+
			"When set, the first line of CSV data is treated as a data row instead of column names.")
	rootCmd.Flags().BoolVarP(&options.SkipErrors, "skip-errors", "s", false,
		"Skip CSV lines that cause parsing errors. By default, errors abort conversion completely.")
	rootCmd.Flags().BoolVar(&logJson, "log-json", false, "Output logs as JSON")
}

func setUpLogs() {
	if logJson {
		logger = log.NewJSONLogger(rootCmd.ErrOrStderr())
	} else {
		logger = log.NewLogfmtLogger(rootCmd.ErrOrStderr())
	}

	if logLevel >= 2 {
		logger = level.NewFilter(logger, level.AllowDebug())
	} else if logLevel == 1 {
		logger = level.NewFilter(logger, level.AllowInfo())
	} else {
		logger = level.NewFilter(logger, level.AllowWarn())
	}
}
