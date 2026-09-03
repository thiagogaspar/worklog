package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"worklog/internal/holidays"
)

var holidaysCmd = &cobra.Command{
	Use:     "holidays",
	Short:   "Listar feriados considerados no planejamento (Curitiba/PR)",
	Example: `  worklog holidays`,
	RunE: func(cmd *cobra.Command, args []string) error {
		now := time.Now()
		list := holidays.All(now)

		fmt.Printf("📅 Feriados %d — Curitiba/PR (obrigatórios, sem pontos facultativos)\n\n", now.Year())
		for _, h := range list {
			marker := "  "
			if sameDay(h.Date, now) {
				marker = "→ "
			} else if h.Date.Before(now) {
				marker = "✓ "
			}
			fmt.Printf("%s%s  %s\n", marker, h.Date.Format("02/01 (Mon)"), h.Name)
		}

		fmt.Println()
		nextWeekPlanned := holidays.PlannedHoursForWeek(now)
		fmt.Printf("Meta da semana atual: %.0fh (ajustada automaticamente por feriado, quando houver)\n", nextWeekPlanned)
		return nil
	},
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func init() {
	rootCmd.AddCommand(holidaysCmd)
}
