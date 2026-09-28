package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"worklog/internal/store"
	"worklog/internal/types"
)

var morningFlags struct{ ticket string; goFlag bool; desc string }

var morningCmd = &cobra.Command{
	Use: "morning", Short: "☀️ Ritual matinal — ontem, tickets, sugestão",
	Example: "  worklog morning\n  worklog morning -t #8606 -g -d \"layout\"",
	RunE: func(cmd *cobra.Command, _ []string) error {
		y := time.Now().AddDate(0, 0, -1)
		ys := y.Format("2006-01-02")

		entries, total, target := loadYesterdayEntries(y, ys)
		fmt.Printf("🌅 %s\n\n📋 Ontem (%s %s)\n", y.Format("Mon, 02 Jan 2006"), y.Weekday().String()[:3], y.Format("02 Jan"))
		if len(entries) == 0 {
			fmt.Printf("   Sem registros — faltam %.0fh\n\n", target)
		} else {
			for _, e := range entries {
				fmt.Printf("   %s  %.1fh", e.Ticket, float64(e.Minutes)/60)
				if e.Desc != "" { fmt.Printf(" — %s", e.Desc) }
				fmt.Println()
			}
			fmt.Printf("   Total: %.1fh de %.0fh", total, target)
			if total < target { fmt.Printf(" — faltam %.1fh", target-total) }
			fmt.Print("\n\n")
		}

		open := fetchOpenTickets()
		yesterdayCommits(y)

		sug := morningFlags.ticket
		if sug == "" && len(open) > 0 { sug = open[0][0] }
		if sug != "" {
			for _, t := range open {
				if t[0] == sug { fmt.Printf("☕ Sugestão: #%s %s\n", t[0], t[1]); break }
			}
		}
		if morningFlags.goFlag && sug != "" { return startSession(sug) }
		return nil
	},
}

func init() {
	morningCmd.Flags().StringVarP(&morningFlags.ticket, "ticket", "t", "", "Ticket")
	morningCmd.Flags().BoolVarP(&morningFlags.goFlag, "go", "g", false, "Iniciar cronômetro")
	morningCmd.Flags().StringVarP(&morningFlags.desc, "desc", "d", "", "Descrição (com --go)")
}

func loadYesterdayEntries(y time.Time, ys string) ([]types.Entry, float64, float64) {
	yw, ww := y.ISOWeek()
	ywStr := fmt.Sprintf("%d-W%02d", yw, ww)
	cy, cw := time.Now().ISOWeek()
	cwStr := fmt.Sprintf("%d-W%02d", cy, cw)
	target := 8.0

	if ywStr == cwStr { s, err := store.Load()
		if err != nil { return nil, 0, target }
		target = s.Week.Planned / 5
		for _, d := range s.Week.Days {
			if d.Date == ys {
				main := mainEntries(d.Entries)
				total := projectEntryTotal(d.Entries, mainProject)
				return main, total, target
			}
		}
		return nil, 0, target
	}
	w, err := store.LoadArchive(ywStr)
	if err != nil { return nil, 0, target }
	target = w.Planned / 5
	for _, d := range w.Days {
		if d.Date == ys {
			main := mainEntries(d.Entries)
			total := projectEntryTotal(d.Entries, mainProject)
			return main, total, target
		}
	}
	return nil, 0, target
}

// milvusChamado é o subconjunto de campos que nos interessa do JSON
// devolvido por `milvus ticket list -f json` (internal/types.Chamado no
// repo do milvus-cli).
type milvusChamado struct {
	Codigo  int    `json:"codigo"`
	Assunto string `json:"assunto"`
	Status  string `json:"status"`
}

func fetchOpenTickets() [][3]string {
	out, err := exec.Command("milvus", "ticket", "list", "-f", "json").Output()
	if err != nil { return nil }
	var chamados []milvusChamado
	if err := json.Unmarshal(out, &chamados); err != nil { return nil }

	var r [][3]string
	skip := map[string]bool{"Finalizado": true, "Cancelado": true}
	for _, c := range chamados {
		if skip[c.Status] { continue }
		r = append(r, [3]string{strconv.Itoa(c.Codigo), c.Assunto, c.Status})
	}
	if len(r) > 0 {
		fmt.Printf("🎫 Tickets abertos (%d)\n", len(r))
		icons := map[string]string{"Atendendo": "⏳", "Pausado": "⏸", "A fazer": "•"}
		for _, t := range r {
			ic := "•"
			if v, ok := icons[t[2]]; ok { ic = v }
			fmt.Printf("   %s #%s  %s  [%s]\n", ic, t[0], t[1], t[2])
		}
		fmt.Println()
	}
	return r
}

func yesterdayCommits(y time.Time) {
	since := y.Format("2006-01-02") + " 00:00"
	unt := time.Now().Format("2006-01-02") + " 00:00"
	git := exec.Command("git", "log", "--since="+since, "--until="+unt, "--format=%ai %s", "--max-count=10")
	git.Dir = "/home/thiago/Work/meioemensagem/www.meioemensagem.com.br/"
	out, err := git.Output()
	fmt.Printf("💻 Commits de ontem\n")
	if err != nil { fmt.Printf("   (nenhum)\n\n"); return }
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") { fmt.Printf("   (nenhum)\n\n"); return }
	for _, l := range lines { fmt.Println("   " + strings.TrimSpace(l)) }
	fmt.Println()
}

func startSession(ticket string) error {
	s, err := store.Load()
	if err != nil { return err }
	if s.Active != nil { fmt.Printf("⚠️  Já ativo: %s\n", s.Active.Ticket); return nil }
	s.Active = &types.ActiveSession{Ticket: ticket, Since: time.Now(), Desc: morningFlags.desc}
	if err := store.Save(s); err != nil { return err }
	fmt.Printf("⏱  #%s — iniciado %s\n", ticket, s.Active.Since.Format("15:04"))
	return nil
}
