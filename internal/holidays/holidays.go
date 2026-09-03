// Package holidays contém a lista de feriados obrigatórios de Curitiba/PR
// (nacionais + municipais, sem pontos facultativos) usada para ajustar
// automaticamente a meta semanal de horas.
package holidays

import "time"

type Holiday struct {
	Date time.Time
	Name string
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

// list2026 — feriados obrigatórios em Curitiba/PR: 9 nacionais + 3 municipais
// (Sexta-feira Santa, Corpus Christi, N. Sra. da Luz dos Pinhais).
// Pontos facultativos (Carnaval, véspera de feriado, etc.) ficam de fora.
var list2026 = []Holiday{
	{date(2026, 1, 1), "Confraternização Universal"},
	{date(2026, 4, 3), "Sexta-feira Santa"},
	{date(2026, 4, 21), "Tiradentes"},
	{date(2026, 5, 1), "Dia do Trabalho"},
	{date(2026, 6, 4), "Corpus Christi"},
	{date(2026, 9, 7), "Independência do Brasil"},
	{date(2026, 9, 8), "Nossa Senhora da Luz dos Pinhais"},
	{date(2026, 11, 2), "Finados"},
	{date(2026, 11, 15), "Proclamação da República"},
	{date(2026, 11, 20), "Consciência Negra"},
	{date(2026, 12, 25), "Natal"},
}

// byYear indexa as listas conhecidas por ano — estender aqui quando o
// calendário do ano seguinte for confirmado.
var byYear = map[int][]Holiday{
	2026: list2026,
}

// All retorna os feriados conhecidos do ano de t, ordenados por data.
func All(t time.Time) []Holiday {
	src := byYear[t.Year()]
	out := make([]Holiday, len(src))
	copy(out, src)
	return out
}

// ForWeek retorna os feriados que caem na semana ISO de t.
func ForWeek(t time.Time) []Holiday {
	y, w := t.ISOWeek()
	var out []Holiday
	for _, h := range byYear[t.Year()] {
		hy, hw := h.Date.ISOWeek()
		if hy == y && hw == w {
			out = append(out, h)
		}
	}
	return out
}

// IsHoliday informa se a data (dia/mês/ano) de t é feriado, e o nome.
func IsHoliday(t time.Time) (bool, string) {
	y, m, d := t.Date()
	for _, h := range byYear[t.Year()] {
		hy, hm, hd := h.Date.Date()
		if hy == y && hm == m && hd == d {
			return true, h.Name
		}
	}
	return false, ""
}

// PlannedHoursForWeek retorna a meta semanal padrão pra semana ISO de t:
// 40h menos 8h pra cada feriado em dia útil (seg-sex) naquela semana.
func PlannedHoursForWeek(t time.Time) float64 {
	planned := 40.0
	for _, h := range ForWeek(t) {
		wd := h.Date.Weekday()
		if wd != time.Saturday && wd != time.Sunday {
			planned -= 8
		}
	}
	if planned < 0 {
		planned = 0
	}
	return planned
}
