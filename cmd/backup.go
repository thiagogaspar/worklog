package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"worklog/internal/store"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Listar backups automáticos ou restaurar um",
	Example: `  worklog backup
  worklog backup restore week-20260817-141903.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		backups, err := store.ListBackups()
		if err != nil {
			return err
		}
		if len(backups) == 0 {
			fmt.Println("📦 Nenhum backup encontrado em ~/.config/worklog/backups/")
			fmt.Println("   Backups são criados automaticamente antes de cada save.")
			return nil
		}
		fmt.Printf("📦 %d backup(s) disponível(is):\n\n", len(backups))
		for _, b := range backups {
			fmt.Printf("  • %s\n", b)
		}
		fmt.Println("\nPara restaurar: worklog backup restore <arquivo>")
		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <arquivo>",
	Short: "Restaurar um backup sobre o week.json atual",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !confirmRestore() {
			fmt.Println("Restauração cancelada.")
			return nil
		}
		if err := store.RestoreBackup(args[0]); err != nil {
			return err
		}
		fmt.Printf("✅ Backup '%s' restaurado.\n", args[0])
		return nil
	},
}

// confirmRestore pede confirmação interativa antes de sobrescrever o week.json.
func confirmRestore() bool {
	fmt.Print("⚠️  Isso vai sobrescrever o week.json atual. Continuar? [s/N] ")
	var resp string
	_, err := fmt.Scanln(&resp)
	if err != nil {
		return false
	}
	return resp == "s" || resp == "S" || resp == "sim" || resp == "SIM"
}

func init() {
	backupCmd.AddCommand(backupRestoreCmd)
	rootCmd.AddCommand(backupCmd)
}
