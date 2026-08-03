package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"worklog/internal/store"
)

var (
	checkDaily   float64
	checkQuiet   bool
	checkNoColor bool
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Verificação rápida de status (para prompt hook)",
	RunE: func(cmd *cobra.Command, args []string) error {
		green, red, yellow, reset := checkColors()

		s, err := store.Load()
		if err != nil {
			return err
		}

		now := time.Now()
		today := now.Format("2006-01-02")
		yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

		var todayTotal, yesterdayTotal float64
		for _, d := range s.Week.Days {
			if d.Date == today {
				todayTotal = d.Total
			}
			if d.Date == yesterday {
				yesterdayTotal = d.Total
			}
		}

		remaining := checkDaily - todayTotal
		yesterdayHadActivity := yesterdayTotal > 0

		// Active session
		if s.Active != nil {
			elapsed := now.Sub(s.Active.Since)
			h := int(elapsed.Hours())
			m := int(elapsed.Minutes()) % 60
			var elapsedStr string
			if h > 0 {
				elapsedStr = fmt.Sprintf("%dh%02dm", h, m)
			} else {
				elapsedStr = fmt.Sprintf("%dm", m)
			}

			if !checkQuiet {
				fmt.Printf("%s⏳ %s — %s ativo%s\n", yellow, s.Active.Ticket, elapsedStr, reset)
			}
			if checkQuiet && remaining > 0 && now.Hour() >= 18 {
				fmt.Printf("%s🔴 Hoje: %.1fh — faltam %.1fh%s\n", red, todayTotal, remaining, reset)
			}
			return nil
		}

		if checkQuiet {
			if todayTotal == 0 {
				if !yesterdayHadActivity {
					fmt.Printf("%s📋 Nenhuma hora registrada hoje%s\n", yellow, reset)
				}
			} else if remaining > 0 && now.Hour() >= 18 {
				fmt.Printf("%s🔴 Hoje: %.1fh — faltam %.1fh%s\n", red, todayTotal, remaining, reset)
			}
			return nil
		}

		switch {
		case todayTotal >= checkDaily:
			fmt.Printf("%s✅ Hoje: %.1fh registradas%s\n", green, todayTotal, reset)
		case remaining > 0 && now.Hour() >= 18:
			fmt.Printf("%s🔴 Hoje: %.1fh — faltam %.1fh%s\n", red, todayTotal, remaining, reset)
		case todayTotal == 0 && !yesterdayHadActivity:
			fmt.Printf("%s📋 Novo dia — nenhuma hora ainda%s\n", yellow, reset)
		case todayTotal < 1:
			fmt.Printf("%s📋 Hoje: %.1fh registradas%s\n", yellow, todayTotal, reset)
		}

		return nil
	},
}

func checkColors() (green, red, yellow, reset string) {
	if checkNoColor {
		return "", "", "", ""
	}
	return "\033[32m", "\033[31m", "\033[33m", "\033[0m"
}

func init() {
	checkCmd.Flags().Float64VarP(&checkDaily, "daily", "d", 8, "Meta diária em horas")
	checkCmd.Flags().BoolVarP(&checkQuiet, "quiet", "q", false, "Mostrar apenas se houver problema")
	checkCmd.Flags().BoolVarP(&checkNoColor, "no-color", "n", false, "Remover cores ANSI")
	rootCmd.AddCommand(checkCmd)
}
