package google

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

type EmailDemand struct {
	From, Subject, Snippet, TicketID string
	IsTicket, IsUrgent               bool
	ReceivedAt                       time.Time
}

var ticketRe = regexp.MustCompile(`\[#(\d+)\]|#(\d{4,})`)

func (c *Client) GetRecentDemands(ctx context.Context) ([]EmailDemand, error) {
	if !c.cfg { return nil, fmt.Errorf("google não configurado") }
	ctx, cancel := context.WithTimeout(ctx, apiTimeout); defer cancel()
	q := "newer_than:2d (chamado OR ticket OR demanda OR solicitacao OR site OR bug OR erro)"
	resp, err := c.gmail.Users.Messages.List("me").Q(q).MaxResults(20).Do()
	if err != nil { return nil, err }

	var r []EmailDemand
	for _, m := range resp.Messages {
		full, _ := c.gmail.Users.Messages.Get("me", m.Id).Fields("snippet", "payload/headers", "internalDate").Do()
		if full == nil { continue }
		d := EmailDemand{Snippet: full.Snippet}
		if full.InternalDate > 0 { d.ReceivedAt = time.Unix(full.InternalDate/1000, 0) }
		for _, h := range full.Payload.Headers {
			if h.Name == "From" { d.From = h.Value }
			if h.Name == "Subject" { d.Subject = h.Value }
		}
		if m := ticketRe.FindStringSubmatch(d.Subject); len(m) > 1 {
			d.IsTicket = true; d.TicketID = m[1]
			if d.TicketID == "" { d.TicketID = m[2] }
		}
		s := d.Subject + " " + d.Snippet
		d.IsUrgent = strings.Contains(strings.ToLower(s), "urgente") || strings.Contains(s, "🔴")
		r = append(r, d)
	}
	sort.Slice(r, func(i, j int) bool { return r[i].ReceivedAt.After(r[j].ReceivedAt) })
	return r, nil
}
