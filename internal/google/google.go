package google

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gauth "google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const (
	gmailScope = "https://www.googleapis.com/auth/gmail.readonly"
	calScope   = "https://www.googleapis.com/auth/calendar.readonly"
	apiTimeout = 30 * time.Second
	userAgent  = "worklog/1"
)

type Client struct {
	gmail    *gmail.Service
	calendar *gauth.Service
	cfg      bool
}

func (c *Client) IsConfigured() bool { return c.cfg }

func NewClient() (*Client, error) {
	cfg, err := LoadConfig()
	if err != nil || !cfg.Enabled { return &Client{cfg: false}, nil }

	oc := &oauth2.Config{
		ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret,
		Endpoint: google.Endpoint, Scopes: []string{gmailScope, calScope},
	}
	tp := cfg.TokenFile; if tp == "" { tp = tokenPath() }
	token, err := LoadToken(tp)
	if err != nil { return nil, fmt.Errorf("token: %w — run worklog config google", err) }

	hc := oc.Client(context.Background(), token)
	g, err := gmail.NewService(context.Background(), option.WithHTTPClient(hc), option.WithUserAgent(userAgent))
	if err != nil { return nil, err }
	c, err := gauth.NewService(context.Background(), option.WithHTTPClient(hc), option.WithUserAgent(userAgent))
	if err != nil { return nil, err }
	return &Client{gmail: g, calendar: c, cfg: true}, nil
}

func Setup(clientID, clientSecret string) error {
	cfg := &Config{Enabled: true, ClientID: clientID, ClientSecret: clientSecret, TokenFile: tokenPath()}
	oc := &oauth2.Config{ClientID: clientID, ClientSecret: clientSecret, Endpoint: google.Endpoint, Scopes: []string{gmailScope, calScope}}

	ln, _ := net.Listen("tcp", "localhost:0"); defer ln.Close()
	oc.RedirectURL = fmt.Sprintf("http://localhost:%d", ln.Addr().(*net.TCPAddr).Port)
	fmt.Printf("\n🔗 Abra no navegador:\n   %s\n\n", oc.AuthCodeURL("s", oauth2.AccessTypeOffline, oauth2.ApprovalForce))

	var code string
	ch := make(chan string, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code = r.URL.Query().Get("code")
		w.Write([]byte("<h1>✅ OK</h1>"))
		ch <- code
	})
	srv := &http.Server{Handler: mux}; go srv.Serve(ln); defer srv.Close()

	select { case code = <-ch: case <-time.After(5 * time.Minute): return fmt.Errorf("timeout") }

	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout); defer cancel()
	token, err := oc.Exchange(ctx, code); if err != nil { return err }
	SaveConfig(cfg); SaveToken(cfg.TokenFile, token)

	hc := oc.Client(ctx, token)
	g, _ := gmail.NewService(ctx, option.WithHTTPClient(hc), option.WithUserAgent(userAgent))
	p, err := g.Users.GetProfile("me").Do(); if err != nil { return err }
	cfg.Email = p.EmailAddress; SaveConfig(cfg)
	fmt.Printf("✅ Google ok — %s\n", p.EmailAddress)
	return nil
}
