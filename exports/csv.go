package exports

import (
	"context"
	"encoding/csv"
	"io"

	dataframe "github.com/rocketlaunchr/dataframe-go"
)

// CSVExportOptions contains options for CSV
type CSVExportOptions struct {
	//optional param to specify what nil values should be encoded
	// as (i.e. NULL, \N, NaN, NA etc)
	NullString *string
	// Range of data subsets to write from dataframe
	Range dataframe.Range
	// Field delimiter (set to ',' by NewWriter)
	Separator rune
	// Set to True to use \r\n as the line terminator
	UseCRLF bool
}

// ExportToCSV exports data object to CSV
func ExportToCSV(ctx context.Context, w io.Writer, df *dataframe.DataFrame, options ...CSVExportOptions) error {

	// Lock Dataframe to
	df.Lock()         // lock dataframe object
	defer df.Unlock() // unlock dataframe

	header := []string{}

	var r dataframe.Range // initial default range r

	nullString := "NaN" // Default will be "NaN"

	cw := csv.NewWriter(w)

	if len(options) > 0 {
		cw.Comma = options[0].Separator
		cw.UseCRLF = options[0].UseCRLF
		r = options[0].Range
		if options[0].NullString != nil {
			nullString = *options[0].NullString
		}
	}

	for _, aSeries := range df.Series {
		header = append(header, aSeries.Name())
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	// DontLock optional parameter is added because df has already been locked above
	if df.NRows(dataframe.Options{DontLock: true}) > 0 {

		s, e, err := r.Limits(df.NRows(dataframe.Options{DontLock: true}))
		if err != nil {
			return err
		}

		refreshCount := 0 // Set up refresh counter
		for row := s; row <= e; row++ {

			// check if error in ctx context
			if err := ctx.Err(); err != nil {
				return err
			}

			refreshCount++
			// flush after every 100 writes
			if refreshCount > 100 { // flush in the 101th count
				cw.Flush()
				if err := cw.Error(); err != nil {
					return err
				}
				refreshCount = 1 // reset refreshCount
			}

			sVals := []string{}
			for _, aSeries := range df.Series {
				val := aSeries.Value(row)                                        // df returns null for empty string fields
				if val == nil || val == "NAN" || val == "nan" || val == "null" { // and NAN for empty number fields
					sVals = append(sVals, nullString)
				} else {
					sVals = append(sVals, aSeries.ValueString(row, dataframe.Options{DontLock: true}))
				}
			}

			// Write every row
			if err := cw.Write(sVals); err != nil {
				return err
			}
		}

	}

	// flush before exit
	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}

	return nil
}
