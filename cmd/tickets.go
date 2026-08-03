package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"worklog/internal/store"
	"worklog/internal/types"
)

var tf = struct{ week bool; ticket, since string }{}

var ticketsCmd = &cobra.Command{
	Use: "tickets", Short: "Horas por ticket (todas as semanas)",
	Example: "  worklog tickets\n  worklog tickets --week\n  worklog tickets --ticket 8562",
	RunE: func(_ *cobra.Command, _ []string) error {
		s, _ := store.Load()
		var since time.Time
		if tf.since != "" { since, _ = time.Parse("2006-01-02", tf.since) }

		var weeks []*types.Week
		if !tf.week {
			arch, _ := store.ListArchive()
			for _, yw := range arch { if w, err := store.LoadArchive(yw); err == nil { weeks = append(weeks, w) } }
		}
		weeks = append(weeks, &s.Week)

		type td struct{ d string; h float64 }
tg := map[string]*tRow{}
		for _, w := range weeks {
			for _, d := range w.Days {
				if !since.IsZero() { if dt, _ := time.Parse("2006-01-02", d.Date); dt.Before(since) { continue } }
				for _, e := range d.Entries {
					if tf.ticket != "" && e.Ticket != tf.ticket { continue }
					r, ok := tg[e.Ticket]
					if !ok { r = &tRow{k: e.Ticket}; tg[e.Ticket] = r }
					h := float64(e.Minutes) / 60
					r.days = append(r.days, td{d.Date, h}); r.t += h
					if r.desc == "" && e.Desc != "" { r.desc = e.Desc }
				}
			}
		}
		if s.Active != nil && (tf.ticket == "" || s.Active.Ticket == tf.ticket) {
			r := tg[s.Active.Ticket]; if r == nil { r = &tRow{k: s.Active.Ticket}; tg[s.Active.Ticket] = r }
			r.t += time.Since(s.Active.Since).Hours(); r.active = true; r.since = s.Active.Since.Format("15:04")
		}
		if len(tg) == 0 { fmt.Println("📊 Nenhum registro"); return nil }

		sorted := make([]tRow, 0, len(tg))
		for _, v := range tg { sorted = append(sorted, *v) }
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].t > sorted[j].t })
		for i := range sorted { sort.Slice(sorted[i].days, func(a, b int) bool { return sorted[i].days[a].d < sorted[i].days[b].d }) }

		if tf.ticket != "" {
			t := sorted[0]
			l := "#" + t.k; if t.desc != "" { l += " — " + t.desc }
			fmt.Printf("📊 %s\n────────────────────\nTotal: %.0fh (%d dias)\n\n", l, t.t, len(t.days))
			for _, d := range t.days { dt, _ := time.Parse("2006-01-02", d.d); fmt.Printf("  %s  %.0fh\n", dt.Format("02/01/2006"), d.h) }
			if t.active { fmt.Printf("\n  ⏳ Ativo desde %s\n", t.since) }
			return nil
		}
		if tf.week { printTickets(sorted, fmt.Sprintf("Semana %s", strings.Replace(s.Week.YearWeek, "-", "", 1)), s.Week.Planned); return nil }
		printTickets(sorted, "Todas as Semanas", 0)
		return nil
	},
}

func init() {
	ticketsCmd.Flags().BoolVarP(&tf.week, "week", "w", false, "Apenas semana atual")
	ticketsCmd.Flags().StringVarP(&tf.ticket, "ticket", "t", "", "Filtrar ticket (ex: 8562)")
	ticketsCmd.Flags().StringVar(&tf.since, "since", "", "Desde YYYY-MM-DD")
	rootCmd.AddCommand(ticketsCmd)
}

type tRow struct{ k string; t float64; desc string; days []td; active bool; since string }

func printTickets(ts []tRow, hdr string, planned float64) {
	fmt.Printf("📊 %s\n────────────────\n", hdr)
	max := 0.0; for _, t := range ts { if t.t > max { max = t.t } }
	total := 0.0
	for _, t := range ts {
		total += t.t
		bar := ""; if max > 0 { bar = strings.Repeat("█", int(t.t/max*20)+1) }
		l := "#" + t.k; if t.desc != "" { if len(t.desc) > 15 { l += " " + t.desc[:15] } else { l += " " + t.desc } }
		fmt.Printf("%-25s %5.0fh  %s\n", l, t.t, bar)
		for _, d := range t.days { dt, _ := time.Parse("2006-01-02", d.d); fmt.Printf("  %s %.0fh\n", dt.Format("02/01"), d.h) }
		fmt.Println()
	}
	fmt.Println("────────────────")
	if planned > 0 { fmt.Printf("Total: %.1fh de %.0fh\n", total, planned) } else { fmt.Printf("Total: %.0fh\n", total) }
}

type td = struct{ d string; h float64 }
