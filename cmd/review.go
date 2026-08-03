package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"worklog/internal/store"
	"worklog/internal/types"
)

var reviewDaily float64

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Revisão detalhada do dia (entradas, ontem, semana)",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.Load()
		if err != nil {
			return err
		}

		now := time.Now()
		today := now.Format("2006-01-02")
		yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

		var todayDay, yesterdayDay *types.Day
		for i := range s.Week.Days {
			if s.Week.Days[i].Date == today {
				todayDay = &s.Week.Days[i]
			}
			if s.Week.Days[i].Date == yesterday {
				yesterdayDay = &s.Week.Days[i]
			}
		}

		fmt.Printf("📅 Revisão — %s\n\n", now.Format("02/01/2006"))

		// Hoje
		fmt.Println("  Hoje:")
		if todayDay != nil && len(todayDay.Entries) > 0 {
			for _, e := range todayDay.Entries {
				h := float64(e.Minutes) / 60
				desc := ""
				if e.Desc != "" {
					desc = "  " + e.Desc
				}
				fmt.Printf("  %s  %s-%s  %.0fh%s\n",
					e.Ticket, e.Start.Format("15:04"), e.End.Format("15:04"), h, desc)
				if len(e.Files) > 0 {
					for _, f := range e.Files {
						fmt.Printf("       📁 %s\n", f)
					}
				}
			}

			overlaps := findOverlaps(todayDay.Entries)
			if len(overlaps) > 0 {
				fmt.Println("\n  ⚠️  Sobreposição detectada:")
				for _, o := range overlaps {
					fmt.Printf("     %s ↔ %s\n", o[0], o[1])
				}
				fmt.Println()
			}

			fmt.Println("  ─────────────────────────")
			remaining := reviewDaily - todayDay.Total
			if remaining > 0 {
				fmt.Printf("  Total: %.1fh de %.0fh — faltam %.1fh\n", todayDay.Total, reviewDaily, remaining)
			} else {
				fmt.Printf("  Total: %.1fh de %.0fh ✅\n", todayDay.Total, reviewDaily)
			}
		} else {
			fmt.Println("  (nenhuma entrada hoje)")
		}
		fmt.Println()

		// Ontem
		fmt.Println("  Ontem:")
		if yesterdayDay != nil && len(yesterdayDay.Entries) > 0 {
			for _, e := range yesterdayDay.Entries {
				h := float64(e.Minutes) / 60
				desc := ""
				if e.Desc != "" {
					desc = "  " + e.Desc
				}
				fmt.Printf("  %s  %s-%s  %.0fh%s\n",
					e.Ticket, e.Start.Format("15:04"), e.End.Format("15:04"), h, desc)
				if len(e.Files) > 0 {
					for _, f := range e.Files {
						fmt.Printf("       📁 %s\n", f)
					}
				}
			}
			fmt.Printf("  Total: %.1fh\n", yesterdayDay.Total)
		} else {
			fmt.Println("  (sem registros)")
		}
		fmt.Println()

		// Semana
		fmt.Printf("  ─── Semana %s ──────────────────\n", s.Week.YearWeek[5:])
		var weekTotal float64
		for _, d := range s.Week.Days {
			fmt.Printf("  %s/%s  %.1fh\n", d.Date[8:10], d.Date[5:7], d.Total)
			weekTotal += d.Total
		}
		weekRemaining := s.Week.Planned - weekTotal
		if weekRemaining > 0 {
			fmt.Printf("  Total: %.1fh de %.0fh — restam %.1fh\n", weekTotal, s.Week.Planned, weekRemaining)
		} else {
			fmt.Printf("  Total: %.1fh de %.0fh ✅\n", weekTotal, s.Week.Planned)
		}

		// Sessão ativa
		if s.Active != nil {
			elapsed := now.Sub(s.Active.Since)
			h := int(elapsed.Hours())
			m := int(elapsed.Minutes()) % 60
			trackStr := ""
			if s.Active.Tracking {
				trackStr = fmt.Sprintf(" 📁 tracking (%d arquivo(s))", len(s.Active.Files))
			}
			fmt.Printf("\n  ⏳ Sessão ativa: %s (desde %s, %dh%02dm)%s\n",
				s.Active.Ticket, s.Active.Since.Format("15:04"), h, m, trackStr)
		}

		return nil
	},
}

func findOverlaps(entries []types.Entry) [][2]string {
	var overlaps [][2]string
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			a, b := entries[i], entries[j]
			if a.Ticket == b.Ticket {
				continue
			}
			if a.Start.Before(b.End) && b.Start.Before(a.End) {
				overlaps = append(overlaps, [2]string{a.Ticket, b.Ticket})
			}
		}
	}
	return overlaps
}

func init() {
	reviewCmd.Flags().Float64VarP(&reviewDaily, "daily", "d", 8, "Meta diária em horas")
	rootCmd.AddCommand(reviewCmd)
}
