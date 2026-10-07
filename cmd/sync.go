package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"worklog/internal/store"
)

const gestaoBase = "https://gestao.antfarmpartners.com"

var sf = struct {
	date     string
	login    bool
	dry      bool
	check    bool
	all      bool
	activity string
	days     int
}{
	activity: "1343",
}

var syncCmd = &cobra.Command{Use: "sync", Short: "Sincronizar com sistemas externos"}
var gestaoCmd = &cobra.Command{Use: "gestao", Short: "→ Antfarm Partners", RunE: runGestao}
var milvusCmd = &cobra.Command{Use: "milvus", Short: "→ Milvus", RunE: runMilvus}

func init() {
	syncCmd.PersistentFlags().StringVarP(&sf.date, "date", "d", "", "Data YYYY-MM-DD (padrao: hoje)")
	syncCmd.PersistentFlags().BoolVar(&sf.login, "login", false, "Login antes de sincronizar")
	syncCmd.PersistentFlags().BoolVar(&sf.dry, "dry-run", false, "Apenas preview")
	syncCmd.PersistentFlags().BoolVar(&sf.check, "check", false, "Gestão: mostrar dia mais recente lançado na atividade")
	syncCmd.PersistentFlags().BoolVar(&sf.all, "all", false, "Gestão --check: listar todos os dias lançados")
	syncCmd.PersistentFlags().StringVar(&sf.activity, "activity", "1343", "Gestão: ID da atividade (padrao: MMID 1343)")
	syncCmd.PersistentFlags().IntVar(&sf.days, "days", 0, "Ultimos N dias")
	syncCmd.AddCommand(gestaoCmd, milvusCmd)
	rootCmd.AddCommand(syncCmd)
}

func syncDates() []string {
	if sf.days > 0 {
		r := make([]string, sf.days)
		for i := 0; i < sf.days; i++ { r[i] = time.Now().AddDate(0, 0, -i).Format("2006-01-02") }
		return r
	}
	if sf.date != "" { return []string{sf.date} }
	return []string{time.Now().Format("2006-01-02")}
}

// entry é uma entrada agregada de qualquer semana (atual ou arquivada).
type entry struct {
	d, t        string
	start, end  string
	h           float64
	desc        string
}

// collectEntries reúne as entries das datas pedidas, buscando na semana atual
// (week.json) e nos archives de semanas anteriores — o sync não fica preso à
// semana corrente.
func collectEntries(dates []string) []entry {
	want := map[string]bool{}
	for _, d := range dates { want[d] = true }

	// Semana atual
	s, _ := store.Load()
	var out []entry
	for _, d := range s.Week.Days {
		if !want[d.Date] { continue }
		for _, e := range d.Entries {
			out = append(out, entry{d.Date, e.Ticket, e.Start.Format("15:04"), e.End.Format("15:04"), float64(e.Minutes) / 60, e.Desc})
		}
	}

	// Semanas anteriores (archive): agrupa datas por YYYY-Www para carregar
	// cada arquivo uma única vez.
	weeks := map[string][]string{}
	for _, d := range dates {
		dt, err := time.Parse("2006-01-02", d)
		if err != nil { continue }
		w := store.WeekOf(dt)
		if w == s.Week.YearWeek { continue } // já coberto acima
		weeks[w] = append(weeks[w], d)
	}
	for wk, ds := range weeks {
		arch, err := store.LoadArchive(wk)
		if err != nil { continue } // semana sem archive
		for _, d := range arch.Days {
			if !want[d.Date] { continue }
			for _, e := range d.Entries {
				out = append(out, entry{d.Date, e.Ticket, e.Start.Format("15:04"), e.End.Format("15:04"), float64(e.Minutes) / 60, e.Desc})
			}
		}
		_ = ds
	}
	return out
}

func runGestao(_ *cobra.Command, _ []string) error {
	if sf.login { if err := gestaoLogin(); err != nil { return err } }
	if _, err := os.Stat("/tmp/gestao_cookies.txt"); os.IsNotExist(err) {
		return fmt.Errorf("sem cookies. Use --login primeiro")
	}
	cl, err := cookieClient()
	if err != nil { return err }

	if sf.check {
		return gestaoCheck(cl)
	}

	// Datas alvo: flags explícitas (--date/--days) têm prioridade; senão usa as
	// últimas 2 semanas úteis (segunda a sexta) como padrão.
	var target []string
	if sf.days > 0 || sf.date != "" {
		target = syncDates()
	} else {
		target = lastWorkWeeks(2)
	}

	// Consulta o Gestão para saber quais dias já estão lançados no período.
	lancados, err := gestaoLancados(cl, target)
	if err != nil { return err }

	// Dias que faltam (não lançados no Gestão).
	var faltando []string
	for _, d := range target {
		if !lancados[d] { faltando = append(faltando, d) }
	}

	if len(faltando) == 0 {
		fmt.Println("✅ Todos os dias do período já estão lançados no Gestão")
		return nil
	}

	entries := collectEntries(faltando)
	if len(entries) == 0 { fmt.Println("Nada a sincronizar (sem registros locais para os dias faltantes)"); return nil }

	fmt.Printf("📋 %d dia(s) faltando, %d entrada(s) local(is)\n", len(faltando), len(entries))
	n := 0
	for _, e := range entries {
		dt, _ := time.Parse("2006-01-02", e.d)
		st := "✅"; if sf.dry { st = "⏭" }
		fmt.Printf("  %s %s %s-%s %.0fh → Gestão (%s)\n", st, e.t, e.start, e.end, e.h, dt.Format("02/01"))
		if sf.dry { n++; continue }
		task := e.desc
		if task != "" { task += " " }
		task += e.t
		body, ct := form(map[string]string{
			"insert": "true", "_request_json": "true", "ID_employee": "100", "ID_client": "213",
			"ID_project": "350", "ID_activity": "1343", "type": "100",
			"date_task": dt.Format("02/01/2006"), "start_and_end": "1",
			"hour_start": e.start, "hour_end": e.end, "task": task,
			"year": strconv.Itoa(dt.Year()), "month": dt.Format("01"), "data_lancamento": dt.Format("02/01/2006"),
		})
		resp, err := cl.Post(gestaoBase+"/Activities/insertHour", ct, body)
		if err != nil || resp.StatusCode != 200 { fmt.Printf("    ⚠️ erro\n"); continue }
		resp.Body.Close()
		n++
	}
	if sf.dry { fmt.Printf("⏭ %d (dry-run)\n", n) } else { fmt.Printf("✅ %d ok\n", n) }
	return nil
}

// lastWorkWeeks retorna os dias úteis (segunda a sexta) das últimas n semanas,
// incluindo a semana atual até hoje. Ex.: lastWorkWeeks(2) → semana atual + anterior.
func lastWorkWeeks(n int) []string {
	now := time.Now()
	var out []string
	cur := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// fim = hoje; se hoje for fim de semana, usa a sexta-feira da semana atual
	fim := cur
	switch fim.Weekday() {
	case time.Saturday: fim = fim.AddDate(0, 0, -1)
	case time.Sunday: fim = fim.AddDate(0, 0, -2)
	}
	// volta até segunda-feira da semana atual
	for cur.Weekday() != time.Monday { cur = cur.AddDate(0, 0, -1) }
	start := cur.AddDate(0, 0, -7*(n-1))
	for d := start; !d.After(fim); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday { continue }
		out = append(out, d.Format("2006-01-02"))
	}
	return out
}

// gestaoLancados pagina a atividade no Gestão e retorna o conjunto de datas já
// lançadas dentro do período alvo. Cada página mostra 10 lançamentos (mais
// recentes primeiro); para o período de 2 semanas bastam as primeiras páginas.
func gestaoLancados(cl *http.Client, target []string) (map[string]bool, error) {
	if len(target) == 0 { return map[string]bool{}, nil }
	minDate, _ := time.Parse("2006-01-02", target[0]) // datas vêm ordenadas (mais antiga primeiro)
	lancados := map[string]bool{}

	// Extrai a data do primeiro lançamento (o mais recente) de cada página.
	pageDate := func(html string) (time.Time, bool) {
		re := regexp.MustCompile(`<td>(\d{2}/\d{2}/\d{4}) \d{2}:\d{2}:\d{2}</td>`)
		if m := re.FindStringSubmatch(html); m != nil {
			t, err := time.Parse("02/01/2006", m[1])
			return t, err == nil
		}
		return time.Time{}, false
	}

	for p := 1; p <= 200; p++ {
		u := fmt.Sprintf("%s/my-activities/detail?id=%s&hours=true&paged=%d", gestaoBase, sf.activity, p)
		resp, err := cl.Get(u)
		if err != nil { return nil, fmt.Errorf("página %d: %w", p, err) }
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		html := string(body)

		if strings.Contains(html, "<title>Login</title>") {
			return nil, fmt.Errorf("sessão expirada. Use --login")
		}

		first, ok := pageDate(html)
		if !ok {
			break // página sem lançamentos → fim
		}
		// Se o lançamento mais recente desta página já é mais antigo que o
		// período alvo, não há mais nada a coletar.
		if first.Before(minDate) { break }

		re := regexp.MustCompile(`<td>(\d{2}/\d{2}/\d{4}) \d{2}:\d{2}:\d{2}</td>`)
		for _, m := range re.FindAllStringSubmatch(html, -1) {
			if _, ok := lancados[m[1]]; ok { continue }
			t, err := time.Parse("02/01/2006", m[1])
			if err != nil { continue }
			if t.Before(minDate) { continue }
			lancados[t.Format("2006-01-02")] = true
		}
	}
	return lancados, nil
}

// gestaoCheck consulta a página da atividade no Gestão. Por padrão mostra o dia
// mais recente lançado; com --all lista todos os dias encontrados.
func gestaoCheck(cl *http.Client) error {
	url := fmt.Sprintf("%s/my-activities/detail?id=%s&hours=true", gestaoBase, sf.activity)
	resp, err := cl.Get(url)
	if err != nil { return fmt.Errorf("consulta Gestão: %w", err) }
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("consulta Gestão: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil { return fmt.Errorf("lendo resposta: %w", err) }

	html := string(body)
	if strings.Contains(html, "<title>Login</title>") {
		return fmt.Errorf("sessão expirada. Use --login")
	}

	// Extrai datas dd/mm/yyyy presentes na página (lançamentos)
	re := regexp.MustCompile(`\b(0[1-9]|[12][0-9]|3[01])/(0[1-9]|1[0-2])/(20\d{2})\b`)
	seen := map[string]bool{}
	var dates []string
	for _, m := range re.FindAllString(html, -1) {
		if seen[m] { continue }
		seen[m] = true
		dates = append(dates, m)
	}

	if len(dates) == 0 {
		fmt.Printf("Atividade %s: nenhum lançamento encontrado na página (ou HTML sem datas reconhecíveis)\n", sf.activity)
		return nil
	}

	sort.Slice(dates, func(i, j int) bool {
		a, _ := time.Parse("02/01/2006", dates[i])
		b, _ := time.Parse("02/01/2006", dates[j])
		return a.Before(b)
	})

	last, _ := time.Parse("02/01/2006", dates[len(dates)-1])
	first, _ := time.Parse("02/01/2006", dates[0])

	fmt.Printf("Atividade %s — %d dias lançados\n", sf.activity, len(dates))
	fmt.Printf("Mais recente: %s (%s)\n", dates[len(dates)-1], relDays(last))

	if sf.all {
		fmt.Printf("Primeiro:     %s (%s)\n", dates[0], relDays(first))
		fmt.Println("── Todos os lançamentos (mais recentes primeiro) ──")
		for i := len(dates) - 1; i >= 0; i-- {
			fmt.Printf("  %s\n", dates[i])
		}
	}

	// Alerta se já cobre até hoje
	today := time.Now().Truncate(24 * time.Hour)
	if last.Equal(today) {
		fmt.Println("✅ Já está lançado até hoje")
	} else {
		fmt.Printf("⚠️ Último lançamento: %s (faltam dias)\n", dates[len(dates)-1])
	}
	return nil
}

// relDays descreve uma data em relação a hoje ("hoje", "ontem", "há N dias").
func relDays(t time.Time) string {
	today := time.Now().Truncate(24 * time.Hour)
	t = t.Truncate(24 * time.Hour)
	d := int(today.Sub(t).Hours() / 24)
	switch {
	case d == 0: return "hoje"
	case d == 1: return "ontem"
	case d > 1:  return fmt.Sprintf("há %d dias", d)
	default:     return fmt.Sprintf("em %d dias", -d)
	}
}

// milvusLancadoHoje devolve os minutos já apontados HOJE no chamado, lidos do
// próprio Milvus (qualquer origem: este sync, `milvus ticket time`, painel).
func milvusLancadoHoje(bin, ticket, today string) (int, error) {
	c := exec.Command(bin, "ticket", "show", ticket, "-f", "json", "--fields", "horas")
	c.Env = append(os.Environ(), "MILVUS_READONLY=1")
	var stderr bytes.Buffer
	c.Stderr = &stderr
	out, err := c.Output()
	if err != nil { return 0, fmt.Errorf("%s", strings.TrimSpace(stderr.String())) }
	var r struct {
		Atendimentos []struct {
			DataInicial string `json:"data_inicial"`
			Total       string `json:"total_horas_atendimento"`
		} `json:"atendimentos"`
	}
	if err := json.Unmarshal(out, &r); err != nil { return 0, fmt.Errorf("resposta ilegível do milvus: %v", err) }
	min := 0
	for _, a := range r.Atendimentos {
		if !strings.HasPrefix(a.DataInicial, today) { continue }
		hm := strings.SplitN(a.Total, ":", 2)
		if len(hm) != 2 { return 0, fmt.Errorf("total de horas inesperado: %q", a.Total) }
		h, e1 := strconv.Atoi(hm[0]); m, e2 := strconv.Atoi(hm[1])
		if e1 != nil || e2 != nil { return 0, fmt.Errorf("total de horas inesperado: %q", a.Total) }
		min += h*60 + m
	}
	return min, nil
}

// runMilvus sincroniza apontamentos de horas do worklog para o Milvus.
// A API do Milvus não aceita data/período no lançamento (entra sempre na
// data de envio) — por isso só entradas de HOJE são enviadas; entradas
// mais antigas são listadas para lançamento manual no painel.
//
// Nunca duplica: antes de enviar, lê do Milvus o que já foi apontado hoje em
// cada chamado e envia só a diferença. Se a leitura falhar, não envia nada
// daquele chamado.
func runMilvus(_ *cobra.Command, _ []string) error {
	home, _ := os.UserHomeDir(); bin := home + "/.local/bin/milvus"
	today := time.Now().Format("2006-01-02")

	type milvusEntry struct {
		d, t, desc string
		min        int
	}
	var entries []milvusEntry
	for _, e := range collectEntries(syncDates()) {
		desc := strings.TrimSpace(e.desc)
		if desc == "" { desc = "Atendimento" }
		entries = append(entries, milvusEntry{e.d, strings.TrimPrefix(e.t, "#"), desc, int(e.h*60 + 0.5)})
	}
	if len(entries) == 0 { fmt.Println("Nada a sincronizar"); return nil }

	// coberto[t] = minutos já no Milvus hoje ainda não abatidos; bloqueado[t] = leitura falhou.
	coberto, bloqueado := map[string]int{}, map[string]bool{}
	lido := map[string]bool{}

	n, manual, pulados, falhas := 0, 0, 0, 0
	for _, e := range entries {
		hs := float64(e.min) / 60
		if e.d != today {
			fmt.Printf("  📋 %s %.1fh (%s) → lançar manualmente no painel (%s)\n", e.t, hs, e.desc, e.d)
			manual++
			continue
		}
		if !lido[e.t] {
			lido[e.t] = true
			m, err := milvusLancadoHoje(bin, e.t, today)
			if err != nil {
				bloqueado[e.t] = true
				fmt.Printf("  ⚠️ %s: não consegui ler as horas já lançadas (%v) — nada enviado para este chamado\n", e.t, err)
			}
			coberto[e.t] = m
		}
		if bloqueado[e.t] { falhas++; continue }

		enviar := e.min
		if c := coberto[e.t]; c > 0 {
			if c >= e.min {
				coberto[e.t] -= e.min
				fmt.Printf("  ⏩ %s %.1fh já está no Milvus hoje — pulado (%s)\n", e.t, hs, e.desc)
				pulados++
				continue
			}
			enviar = e.min - c
			coberto[e.t] = 0
			fmt.Printf("  ↪ %s: %.2fh de %.2fh já estão no Milvus — enviando só a diferença\n", e.t, float64(c)/60, hs)
		}

		st := "✅"; if sf.dry { st = "⏭" }
		fmt.Printf("  %s %s %.2fh → Milvus (%s)\n", st, e.t, float64(enviar)/60, e.desc)
		if sf.dry { n++; continue }
		out, err := exec.Command(bin, "ticket", "time", e.t, "-H", fmt.Sprintf("%.2f", float64(enviar)/60), "-m", e.desc, "-y").CombinedOutput()
		if err != nil {
			// Escrita com erro pode ter sido gravada: não tentar mais nada neste chamado nesta execução.
			bloqueado[e.t] = true; falhas++
			fmt.Printf("    ⚠️ %s\n", strings.TrimSpace(string(out)))
			continue
		}
		n++
	}
	if sf.dry { fmt.Printf("⏭ %d a enviar (dry-run)", n) } else { fmt.Printf("✅ %d ok", n) }
	if pulados > 0 { fmt.Printf("  ·  ⏩ %d já lançado(s)", pulados) }
	if falhas > 0 { fmt.Printf("  ·  ⚠️ %d não enviado(s)", falhas) }
	if manual > 0 { fmt.Printf("  ·  📋 %d retroativo(s) para lançar manualmente", manual) }
	fmt.Println()
	return nil
}

func gestaoLogin() error {
	r := bufio.NewReader(os.Stdin)
	fmt.Print("Email: "); email, _ := r.ReadString('\n')
	fmt.Print("Senha: "); pass, _ := r.ReadString('\n')
	email = strings.TrimSpace(email); pass = strings.TrimSpace(pass)
	if email == "" || pass == "" { return fmt.Errorf("email/senha obrigatorios") }
	body, ct := form(map[string]string{"email": email, "password": pass, "_request_json": "true", "login": "true"})
	resp, err := http.Post(gestaoBase+"/Login/Send", ct, body)
	if err != nil { return err }; defer resp.Body.Close()
	if resp.StatusCode != 200 { b, _ := io.ReadAll(resp.Body); return fmt.Errorf("login falhou: %s", string(b)) }
	var lines []string
	lines = append(lines, "# Netscape HTTP Cookie File")
	for _, c := range resp.Cookies() {
		dom, pth := c.Domain, c.Path
		if dom == "" { dom = ".gestao.antfarmpartners.com" }
		if pth == "" { pth = "/" }
		sec := "FALSE"; if c.Secure { sec = "TRUE" }
		exp := c.Expires.Unix()
		if c.MaxAge > 0 { exp = time.Now().Add(time.Duration(c.MaxAge) * time.Second).Unix() }
		lines = append(lines, fmt.Sprintf("%s\tTRUE\t%s\t%s\t%d\t%s\t%s", dom, pth, sec, exp, c.Name, c.Value))
	}
	os.WriteFile("/tmp/gestao_cookies.txt", []byte(strings.Join(lines, "\n")+"\n"), 0600)
	fmt.Println("✅ Login ok"); return nil
}

func cookieClient() (*http.Client, error) {
	jar, _ := cookiejar.New(nil); data, _ := os.ReadFile("/tmp/gestao_cookies.txt")
	var ck []*http.Cookie
	for _, l := range strings.Split(string(data), "\n") {
		l = strings.TrimSpace(l)
		if l == "" || l[0] == '#' { continue }
		p := strings.Split(l, "\t")
		if len(p) < 7 { continue }
		exp, _ := strconv.ParseInt(p[4], 10, 64)
		ck = append(ck, &http.Cookie{Domain: p[0], Path: p[2], Secure: p[3] == "TRUE", Expires: time.Unix(exp, 0), Name: p[5], Value: p[6]})
	}
	u, _ := url.Parse(gestaoBase); jar.SetCookies(u, ck)
	return &http.Client{Jar: jar}, nil
}

func form(f map[string]string) (io.Reader, string) {
	var b bytes.Buffer; w := multipart.NewWriter(&b)
	for k, v := range f { w.WriteField(k, v) }
	w.Close(); return &b, w.FormDataContentType()
}
