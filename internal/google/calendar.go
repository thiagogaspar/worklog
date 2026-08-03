package google

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type CalendarEvent struct {
	Title, Location, MeetLink string
	StartTime, EndTime        time.Time
	IsAllDay, HasMeetLink     bool
}

func (c *Client) GetTodayEvents(ctx context.Context) ([]CalendarEvent, error) {
	if !c.cfg { return nil, fmt.Errorf("google não configurado") }
	n := time.Now()
	s := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location())
	e := s.Add(24 * time.Hour)
	ctx, cancel := context.WithTimeout(ctx, apiTimeout); defer cancel()
	events, err := c.calendar.Events.List("primary").TimeMin(s.Format(time.RFC3339)).
		TimeMax(e.Format(time.RFC3339)).OrderBy("startTime").SingleEvents(true).MaxResults(50).Do()
	if err != nil { return nil, err }

	r := make([]CalendarEvent, 0, len(events.Items))
	for _, ev := range events.Items {
		ce := CalendarEvent{Title: ev.Summary, Location: ev.Location}
		if ev.Start != nil {
			if ev.Start.Date != "" { ce.IsAllDay = true; ce.StartTime, _ = time.Parse("2006-01-02", ev.Start.Date) } else { ce.StartTime, _ = time.Parse(time.RFC3339, ev.Start.DateTime) }
		}
		if ev.End != nil {
			if ev.End.DateTime != "" { ce.EndTime, _ = time.Parse(time.RFC3339, ev.End.DateTime) } else { ce.EndTime, _ = time.Parse("2006-01-02", ev.End.Date) }
		}
		if ev.ConferenceData != nil {
			for _, ep := range ev.ConferenceData.EntryPoints {
				if strings.Contains(ep.Uri, "meet.google.com") { ce.HasMeetLink, ce.MeetLink = true, ep.Uri; break }
			}
		}
		if !ce.HasMeetLink && ev.HangoutLink != "" { ce.HasMeetLink, ce.MeetLink = true, ev.HangoutLink }
		r = append(r, ce)
	}
	sort.Slice(r, func(i, j int) bool { return r[i].StartTime.Before(r[j].StartTime) })
	return r, nil
}
