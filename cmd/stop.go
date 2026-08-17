package cmd

import (
	"fmt"
	"math"
	"time"

	"github.com/spf13/cobra"
	"worklog/internal/store"
	"worklog/internal/types"
)

const maxSessionMinutes = 8 * 60 // sessões são sempre de 8h

var stopDesc string

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Parar cronômetro e registrar tempo",
	Example: `  worklog stop
  worklog stop -d "diagnostico e deploy"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.Load()
		if err != nil {
			return err
		}
		if s.Active == nil {
			fmt.Println("Nenhuma sessão ativa. Use 'worklog start <ticket>' primeiro.")
			return nil
		}

		now := time.Now()
		elapsed := int(now.Sub(s.Active.Since).Minutes())
		// Sessões do projeto fixo são sempre de 8h: se ficou aberta do dia
		// anterior (ou mais), o stop registra no máximo 8h. Outros projetos
		// registram o tempo real decorrido (controle local pontual).
		minutes := elapsed
		capped := ""
		if isMainProject(s.Active.Project) {
			minutes = int(math.Min(float64(elapsed), maxSessionMinutes))
			if minutes == maxSessionMinutes && elapsed > maxSessionMinutes {
				capped = " (cap 8h)"
			}
		}

		ticket := s.Active.Ticket
		since := s.Active.Since
		trackedFiles := s.Active.Files
		project := s.Active.Project

		// Entrada cai no dia em que a sessão começou (ex.: sessão de ontem
		// parada hoje registra em ontem, com 8h).
		dayIdx := store.FindOrCreateDayAt(s, since)
		s.Week.Days[dayIdx].Entries = append(s.Week.Days[dayIdx].Entries, types.Entry{
			Ticket:  ticket,
			Project: project,
			Start:   since,
			End:     since.Add(time.Duration(minutes) * time.Minute),
			Minutes: minutes,
			Desc:    stopDesc,
			Files:   trackedFiles,
		})
		s.Week.Days[dayIdx].Total += float64(minutes) / 60

		s.Active = nil

		if err := store.Save(s); err != nil {
			return err
		}

		fileInfo := ""
		if len(trackedFiles) > 0 {
			fileInfo = fmt.Sprintf(" 📁 %d arquivo(s)", len(trackedFiles))
		}
		projInfo := ""
		if !isMainProject(project) {
			projInfo = fmt.Sprintf(" [%s]", projectLabel(project))
		}
		fmt.Printf("⏱  %s — %.1fh (encerrado às %s)%s%s%s\n", ticket, float64(minutes)/60, now.Format("15:04"), projInfo, capped, fileInfo)
		return nil
	},
}

func init() {
	stopCmd.Flags().StringVarP(&stopDesc, "desc", "d", "", "Descricao do que foi feito")
}
