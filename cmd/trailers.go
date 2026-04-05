package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/defer-ai/cli/internal/decision"
	"github.com/spf13/cobra"
)

var trailersIDs string

var trailersCmd = &cobra.Command{
	Use:   "trailers",
	Short: "Print Decision-Ref git trailers for decided decisions",
	Long: `Print git trailer lines for all decided decisions in the current project.

Each decided decision produces a line like:
  Decision-Ref: @STA-0001

Use --ids to filter to specific decision IDs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		store, err := decision.LoadStore(cwd)
		if err != nil {
			return err
		}
		if store == nil {
			return nil // no store, no trailers — exit 0
		}

		var out string
		if trailersIDs != "" {
			ids := strings.Split(trailersIDs, ",")
			for i := range ids {
				ids[i] = strings.TrimSpace(ids[i])
			}
			out = decision.TrailersForIDs(store, ids)
		} else {
			out = decision.Trailers(store)
		}

		if out != "" {
			fmt.Println(out)
		}
		return nil
	},
}

func init() {
	trailersCmd.Flags().StringVar(&trailersIDs, "ids", "", "Comma-separated decision IDs to filter (e.g. STA-0001,DAT-0001)")
	rootCmd.AddCommand(trailersCmd)
}
