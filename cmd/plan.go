package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"worklog/internal/store"
)

var planCmd = &cobra.Command{
	Use:   "plan <horas>",
	Short: "Definir horas planejadas na semana (padrão: 40)",
	Args:  cobra.ExactArgs(1),
	Example: `  worklog plan 40
  worklog plan 30`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.Load()
		if err != nil {
			return err
		}

		var h float64
		if _, err := fmt.Sscanf(args[0], "%f", &h); err != nil || h <= 0 || h > 168 {
			return fmt.Errorf("horas invalidas: use um numero entre 1 e 168")
		}

		s.Week.Planned = h
		if err := store.Save(s); err != nil {
			return err
		}

		fmt.Printf("📅 Semana %s — planejado: %.0fh\n", s.Week.YearWeek, h)
		return nil
	},
}
