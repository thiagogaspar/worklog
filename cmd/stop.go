package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"worklog/internal/store"
	"worklog/internal/types"
)

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
		minutes := int(now.Sub(s.Active.Since).Minutes())
		hours := float64(minutes) / 60

		ticket := s.Active.Ticket
		since := s.Active.Since

		dayIdx := store.FindOrCreateDay(s)
		s.Week.Days[dayIdx].Entries = append(s.Week.Days[dayIdx].Entries, types.Entry{
			Ticket:  ticket,
			Start:   since,
			End:     now,
			Minutes: minutes,
			Desc:    stopDesc,
		})
		s.Week.Days[dayIdx].Total += hours

		s.Active = nil

		if err := store.Save(s); err != nil {
			return err
		}

		fmt.Printf("⏱  %s — %.1fh (encerrado às %s)\n", ticket, hours, now.Format("15:04"))
		return nil
	},
}

func init() {
	stopCmd.Flags().StringVarP(&stopDesc, "desc", "d", "", "Descricao do que foi feito")
}
