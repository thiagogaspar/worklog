package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"worklog/internal/store"
)

const gestaoBase = "https://gestao.antfarmpartners.com"

var sf = struct{ date string; login, dry bool; days int }{}

var syncCmd = &cobra.Command{Use: "sync", Short: "Sincronizar com sistemas externos"}
var gestaoCmd = &cobra.Command{Use: "gestao", Short: "→ Antfarm Partners", RunE: runGestao}
var movideskCmd = &cobra.Command{Use: "movidesk", Short: "→ Movidesk", RunE: runMovidesk}

func init() {
	syncCmd.PersistentFlags().StringVarP(&sf.date, "date", "d", "", "Data YYYY-MM-DD (padrao: hoje)")
	syncCmd.PersistentFlags().BoolVar(&sf.login, "login", false, "Login antes de sincronizar")
	syncCmd.PersistentFlags().BoolVar(&sf.dry, "dry-run", false, "Apenas preview")
	syncCmd.PersistentFlags().IntVar(&sf.days, "days", 0, "Ultimos N dias")
	syncCmd.AddCommand(gestaoCmd, movideskCmd)
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

func runGestao(_ *cobra.Command, _ []string) error {
	if sf.login { if err := gestaoLogin(); err != nil { return err } }
	if _, err := os.Stat("/tmp/gestao_cookies.txt"); os.IsNotExist(err) {
		return fmt.Errorf("sem cookies. Use --login primeiro")
	}
	cl, err := cookieClient()
	if err != nil { return err }
	s, _ := store.Load()
	ds := syncDates()

	se := map[string]bool{}
	for _, d := range ds { se[d] = true }

	var entries []struct{ d string; t string; start, end string; h float64; desc string }
	for _, d := range s.Week.Days {
		if !se[d.Date] { continue }
		for _, e := range d.Entries {
			entries = append(entries, struct{ d string; t string; start, end string; h float64; desc string }{
				d.Date, e.Ticket, e.Start.Format("15:04"), e.End.Format("15:04"), float64(e.Minutes) / 60, e.Desc,
			})
		}
	}
	if len(entries) == 0 { fmt.Println("Nada a sincronizar"); return nil }

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

func runMovidesk(_ *cobra.Command, _ []string) error {
	s, _ := store.Load()
	ds := syncDates(); se := map[string]bool{}
	for _, d := range ds { se[d] = true }
	home, _ := os.UserHomeDir(); bin := home + "/.local/bin/movidesk"
	var entries []struct{ d, t string; h float64 }
	for _, d := range s.Week.Days {
		if !se[d.Date] { continue }
		for _, e := range d.Entries {
			entries = append(entries, struct{ d, t string; h float64 }{
				d.Date, strings.TrimPrefix(e.Ticket, "#"), float64(e.Minutes) / 60,
			})
		}
	}
	if len(entries) == 0 { fmt.Println("Nada a sincronizar"); return nil }
	n := 0
	for _, e := range entries {
		st := "✅"; if sf.dry { st = "⏭" }
		fmt.Printf("  %s %s %.1fh → Movidesk (%s)\n", st, e.t, e.h, e.d[5:])
		if sf.dry { n++; continue }
		out, err := exec.Command(bin, "ticket", "time", e.t, "-H", fmt.Sprintf("%.1f", e.h), "-d", e.d, "-y").CombinedOutput()
		if err != nil { fmt.Printf("    ⚠️ %s\n", strings.TrimSpace(string(out))); continue }
		n++
	}
	if sf.dry { fmt.Printf("⏭ %d (dry-run)\n", n) } else { fmt.Printf("✅ %d ok\n", n) }
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
