package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "worklog",
	Short: "Controle de horas por demanda",
	Long: `Registra o tempo gasto em chamados e tarefas.

Armazena localmente em ~/.config/worklog/week.json.
Cada semana tem 40h planejadas. Use start/stop pra cronometrar.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(morningCmd)
}
