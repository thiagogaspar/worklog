package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"worklog/internal/store"
	"worklog/internal/types"
)

var logFlags struct {
	desc    string
	date    string
	project string
}

var logCmd = &cobra.Command{
	Use:     "log <ticket> <horas>",
	Short:   "Registrar horas manualmente sem cronômetro",
	Args:    cobra.ExactArgs(2),
	Example: `  worklog log #8278 2
  worklog log #8279 3 -d "imagens ces"
  worklog log #8277 3 -d "music branding" --date 2026-07-09`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ticket := args[0]

		var hours float64
		if _, err := fmt.Sscanf(args[1], "%f", &hours); err != nil {
			return fmt.Errorf("horas inválidas: %s", args[1])
		}
		if hours <= 0 {
			return fmt.Errorf("horas deve ser > 0")
		}

		dateStr := logFlags.date
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}
		start, err := time.Parse("2006-01-02 15:04", dateStr+" 09:00")
		if err != nil {
			return fmt.Errorf("data inválida: %s", dateStr)
		}
		end := start.Add(time.Duration(hours * float64(time.Hour)))
		minutes := int(hours * 60)

		// Projeto: flag -p tem prioridade; senão detecta pelo git remote.
		project := logFlags.project
		if project == "" {
			project = detectProject()
		}

		targetWeek := store.WeekOf(start)
		currentWeek := store.WeekOf(time.Now())

		if targetWeek != currentWeek {
			// Semana passada: carrega do archive, modifica, salva de volta
			w, err := store.LoadArchive(targetWeek)
			if err != nil {
				return fmt.Errorf("semana %s não encontrada no archive: %w", targetWeek, err)
			}

			var dayIdx int
			found := false
			for i, d := range w.Days {
				if d.Date == dateStr {
					dayIdx = i
					found = true
					break
				}
			}
			if !found {
				w.Days = append(w.Days, types.Day{Date: dateStr})
				dayIdx = len(w.Days) - 1
			}

			w.Days[dayIdx].Entries = append(w.Days[dayIdx].Entries, types.Entry{
				Ticket:  ticket,
				Project: project,
				Start:   start,
				End:     end,
				Minutes: minutes,
				Desc:    logFlags.desc,
			})
			w.Days[dayIdx].Total += hours

			if err := store.SaveArchive(w); err != nil {
				return err
			}
			fmt.Printf("📝  %s — %.1fh registrado em %s (archive %s)\n", ticket, hours, dateStr, targetWeek)
		} else {
			// Semana atual: usa o fluxo normal via store.Load
			s, err := store.Load()
			if err != nil {
				return err
			}

			var dayIdx int
			found := false
			for i, d := range s.Week.Days {
				if d.Date == dateStr {
					dayIdx = i
					found = true
					break
				}
			}
			if !found {
				s.Week.Days = append(s.Week.Days, types.Day{Date: dateStr})
				dayIdx = len(s.Week.Days) - 1
			}

			s.Week.Days[dayIdx].Entries = append(s.Week.Days[dayIdx].Entries, types.Entry{
				Ticket:  ticket,
				Project: project,
				Start:   start,
				End:     end,
				Minutes: minutes,
				Desc:    logFlags.desc,
			})
			s.Week.Days[dayIdx].Total += hours

			if err := store.Save(s); err != nil {
				return err
			}
			fmt.Printf("📝  %s — %.1fh registrado em %s\n", ticket, hours, dateStr)
		}

		if logFlags.desc != "" {
			fmt.Printf("   → %s\n", logFlags.desc)
		}
		if !isMainProject(project) {
			fmt.Printf("   → projeto: %s\n", projectLabel(project))
		}
		return nil
	},
}

func init() {
	logCmd.Flags().StringVarP(&logFlags.desc, "desc", "d", "", "Descricao do que foi feito")
	logCmd.Flags().StringVarP(&logFlags.date, "date", "", "", "Data (YYYY-MM-DD, padrao: hoje)")
	logCmd.Flags().StringVarP(&logFlags.project, "project", "p", "", "Projeto (padrao: detecta via git remote)")
	rootCmd.AddCommand(logCmd)
}
