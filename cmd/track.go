package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"worklog/internal/store"
)

var (
	trackList   bool
	trackRemove string
)

var trackCmd = &cobra.Command{
	Use:   "track [file...]",
	Short: "Registrar arquivos alterados no chamado ativo (--track no start)",
	Long: `Registra quais arquivos foram alterados durante a sessão ativa.
Os arquivos são salvos junto com a entrada no week.json.

Use --track no worklog start para ativar o rastreamento automaticamente.
`,
	Example: `  worklog track includes/modal.php css/default.css
  worklog track --remove includes/modal.php
  worklog track --list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.Load()
		if err != nil {
			return err
		}

		if s.Active == nil {
			return fmt.Errorf("nenhuma sessão ativa. Use 'worklog start #ticket' primeiro.")
		}

		// Ativa tracking automaticamente se estava desligado
		if !s.Active.Tracking {
			s.Active.Tracking = true
		}

		// Modo lista
		if trackList {
			if len(s.Active.Files) == 0 {
				fmt.Printf("📁  %s — nenhum arquivo registrado\n", s.Active.Ticket)
				return nil
			}
			fmt.Printf("📁  %s — %d arquivo(s):\n", s.Active.Ticket, len(s.Active.Files))
			for _, f := range s.Active.Files {
				fmt.Printf("   %s\n", f)
			}
			return nil
		}

		// Modo remove
		if trackRemove != "" {
			newFiles := make([]string, 0, len(s.Active.Files))
			found := false
			for _, f := range s.Active.Files {
				if f == trackRemove {
					found = true
					continue
				}
				newFiles = append(newFiles, f)
			}
			if !found {
				return fmt.Errorf("arquivo '%s' não encontrado na sessão atual", trackRemove)
			}
			s.Active.Files = newFiles
			if err := store.Save(s); err != nil {
				return err
			}
			fmt.Printf("🗑  removido: %s (%d arquivo(s) restantes)\n", trackRemove, len(s.Active.Files))
			return nil
		}

		// Modo add
		if len(args) == 0 {
			return fmt.Errorf("informe ao menos um arquivo, ou use --list / --remove")
		}

		added := 0
		for _, a := range args {
			f := filepath.Clean(a)
			// Evita duplicatas
			dup := false
			for _, existing := range s.Active.Files {
				if existing == f {
					dup = true
					break
				}
			}
			if !dup {
				s.Active.Files = append(s.Active.Files, f)
				added++
			}
		}

		if err := store.Save(s); err != nil {
			return err
		}

		plural := ""
		if added > 1 {
			plural = "s"
		}
		fmt.Printf("📁  +%d arquivo%s em %s (%d total)\n", added, plural, s.Active.Ticket, len(s.Active.Files))
		return nil
	},
}

func init() {
	trackCmd.Flags().BoolVarP(&trackList, "list", "l", false, "Listar arquivos registrados na sessão ativa")
	trackCmd.Flags().StringVarP(&trackRemove, "remove", "r", "", "Remover arquivo da lista")
	rootCmd.AddCommand(trackCmd)
}
