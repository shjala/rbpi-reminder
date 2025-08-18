package main

import (
	"context"
	"fmt"
	"time"

	"github.com/emersion/go-ical"
	webdav "github.com/jonyTF/go-webdav"
	"github.com/jonyTF/go-webdav/caldav"
)

type CalendarEvent struct {
	ID          string
	StartTime   time.Time
	EndTime     time.Time
	TimeZone    string
	Description string
}

type calendarSession struct {
	ctx        context.Context
	cDavClient *caldav.Client
	calendars  []caldav.Calendar
}

// Event represents the structure of the event details.
func (e CalendarEvent) toString() string {
	if e.StartTime.IsZero() && e.EndTime.IsZero() {
		return fmt.Sprintf("Task '%s' is scheduled for today", e.Description)
	}

	if e.EndTime.IsZero() {
		return fmt.Sprintf("Task '%s' is scheduled for whole day, today at %s",
			e.Description,
			e.StartTime.Format("3:04 PM"))
	}

	// Get the duration in a human-readable format
	duration := e.EndTime.Sub(e.StartTime)
	durationStr := formatDuration(duration)

	isOrWas := "is"
	if e.EndTime.Before(time.Now()) {
		isOrWas = "was"
	}

	return fmt.Sprintf(
		"Task \"%s\" %s scheduled for today for %s, from %s to %s",
		e.Description,
		isOrWas,
		durationStr,
		e.StartTime.Format("3:04 PM"),
		e.EndTime.Format("3:04 PM"))
}

func getTodayCalEvents() []CalendarEvent {
	now := time.Now()
	start := startOfDay(now)
	end := endOfDay(now)
	return getCalEvents(start, end)
}

func getCalendarSession() (*calendarSession, error) {
	ctx := context.Background()
	client := webdav.HTTPClientWithBasicAuth(nil,
		SysSecrets.IcloudConfig.Username,
		SysSecrets.IcloudConfig.AppSpecificPassword)

	wDAV, err := webdav.NewClient(client, SysSecrets.IcloudConfig.CalDAVBaseUrl)
	if err != nil {
		return nil, fmt.Errorf("error creating webdav client: %v", err)
	}
	principal, err := wDAV.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return nil, fmt.Errorf("error finding current user principal: %v", err)
	}
	cDAV, err := caldav.NewClient(client, SysSecrets.IcloudConfig.CalDAVBaseUrl)
	if err != nil {
		return nil, fmt.Errorf("error creating caldav client: %v", err)
	}
	calHome, err := cDAV.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return nil, fmt.Errorf("error finding calendar home set: %v", err)
	}
	cals, err := cDAV.FindCalendars(ctx, calHome)
	if err != nil {
		return nil, fmt.Errorf("error finding calendars: %v", err)
	}

	session := &calendarSession{
		ctx:        ctx,
		cDavClient: cDAV,
		calendars:  cals,
	}
	return session, nil
}

func getCalEvents(start, end time.Time) []CalendarEvent {
	ss, err := getCalendarSession()
	if err != nil {
		logError("error finding calendars: %v", err)
		return []CalendarEvent{}
	}

	query := &caldav.CalendarQuery{
		CompRequest: caldav.CalendarCompRequest{
			Name: "VCALENDAR",
			Comps: []caldav.CalendarCompRequest{{
				Name: "VEVENT",
				Props: []string{
					"SUMMARY",
					"UID",
					"DTSTART",
					"DTEND",
					"DURATION",
				},
			}},
			Expand: &caldav.CalendarExpandRequest{
				Start: start,
				End:   end,
			},
		},
		CompFilter: caldav.CompFilter{
			Name: "VCALENDAR",
			Comps: []caldav.CompFilter{{
				Name:  "VEVENT",
				Start: start,
				End:   end,
			}},
		},
	}

	allEvents := []CalendarEvent{}
	for _, cal := range ss.calendars {
		calQuery, err := ss.cDavClient.QueryCalendar(ss.ctx, cal.Path, query)
		if err != nil {
			logError("failed to query events for cal: %s", cal.Path)
			continue
		}

		events := getEventsFromCalQuery(calQuery)
		allEvents = append(allEvents, events...)
	}

	return allEvents
}

func getEventsFromCalQuery(events []caldav.CalendarObject) []CalendarEvent {
	calEvents := []CalendarEvent{}
	for _, event := range events {
		e := event.Data.Events()
		for _, ev := range e {
			dtStart := ev.Props.Get("DTSTART")
			dtEnd := ev.Props.Get("DTEND")
			if dtStart == nil || dtEnd == nil {
				continue
			}

			id := ev.Props.Get("UID").Value
			start := eventTimeToTime(dtStart)
			end := eventTimeToTime(dtEnd)
			calEvents = append(calEvents, CalendarEvent{
				ID:          id,
				StartTime:   start,
				EndTime:     end,
				TimeZone:    getCalEventTimeZone(dtStart).String(),
				Description: ev.Props.Get("SUMMARY").Value,
			})
		}
	}

	return calEvents
}

func eventTimeToTime(eventTime *ical.Prop) time.Time {
	t, err := time.ParseInLocation("20060102T150405", eventTime.Value, getCalEventTimeZone(eventTime))
	if err != nil {
		return time.Time{}
	}
	return t
}

func getCalEventTimeZone(val *ical.Prop) *time.Location {
	if val == nil {
		return time.Local
	}

	loc, err := time.LoadLocation(val.Params.Get("TZID"))
	if err != nil {
		logError("failed to load timezone %s: %v", val.Params.Get("TZID"), err)
		return time.Local
	}

	return loc
}
