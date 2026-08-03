package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"worklog/internal/store"
	"worklog/internal/types"
)

var reportWeek string

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Mostrar resumo da semana (ou --week para semanas anteriores)",
	Example: `  worklog report
  worklog report --week 2026-W27`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if reportWeek != "" {
			w, err := store.LoadArchive(reportWeek)
			if err != nil {
				return err
			}
			printWeek(w, nil)
			return nil
		}
		s, err := store.Load()
		if err != nil {
			return err
		}
		printWeek(&s.Week, s.Active)
		return nil
	},
}

func init() {
	reportCmd.Flags().StringVarP(&reportWeek, "week", "w", "", "Semana arquivada (ex: 2026-W27)")
}

func printWeek(week *types.Week, active *types.ActiveSession) {
	fmt.Printf("📅 Semana %s\n", week.YearWeek)
	fmt.Printf("   Planejado: %.0fh\n", week.Planned)

	var total float64
	for _, d := range week.Days {
		fmt.Printf("\n  ▸ %s (%.1fh)\n", d.Date, d.Total)
		for _, e := range d.Entries {
			h := float64(e.Minutes) / 60
			desc := ""
			if e.Desc != "" {
				desc = " — " + e.Desc
			}
			fmt.Printf("    • %s: %.1fh%s\n", e.Ticket, h, desc)
		}
		total += d.Total
	}

	fmt.Printf("\n  ─────────────────\n")
	fmt.Printf("  Total:  %.1fh de %.0fh\n", total, week.Planned)
	if total < week.Planned {
		fmt.Printf("  Restam: %.1fh\n", week.Planned-total)
	} else {
		fmt.Printf("  Excedente: %.1fh\n", total-week.Planned)
	}

	if active != nil {
		fmt.Printf("\n  ⏳ Sessão ativa: %s (desde %s)\n", active.Ticket, active.Since.Format("15:04"))
	}
}
