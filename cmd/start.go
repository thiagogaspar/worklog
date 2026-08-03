package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"worklog/internal/store"
	"worklog/internal/types"
)

var startDesc string
var startTrack bool

var startCmd = &cobra.Command{
	Use:   "start <ticket>",
	Short: "Iniciar cronômetro para uma demanda",
	Args:  cobra.ExactArgs(1),
	Example: `  worklog start #8510
  worklog start "CES migration"
  worklog start #8562 --track`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.Load()
		if err != nil {
			return err
		}

		if s.Active != nil {
			fmt.Printf("Já há sessão ativa: %s (desde %s)\n", s.Active.Ticket, s.Active.Since.Format("15:04"))
			fmt.Println("Use 'worklog stop' antes de iniciar outra.")
			return nil
		}

		s.Active = &types.ActiveSession{
			Ticket:   args[0],
			Since:    time.Now(),
			Desc:     startDesc,
			Tracking: startTrack,
		}

		if err := store.Save(s); err != nil {
			return err
		}

		trackInfo := ""
		if startTrack {
			trackInfo = " 📁 tracking ativo"
		}
		fmt.Printf("⏱  %s — iniciado às %s%s\n", args[0], s.Active.Since.Format("15:04"), trackInfo)
		return nil
	},
}

func init() {
	startCmd.Flags().StringVarP(&startDesc, "desc", "d", "", "Descrição inicial da tarefa")
	startCmd.Flags().BoolVarP(&startTrack, "track", "t", false, "Ativar rastreamento de arquivos alterados")
}
