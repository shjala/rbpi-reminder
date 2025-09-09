package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

var syncEvent sync.Mutex

type LocalEvent struct {
	Event               CalendarEvent
	StartAnnounced      bool
	CheckStartAnnounced bool
	EndAnnounced        bool
	LastTimeReminded    time.Time
}

// saveEventLocally saves the event to the local storage.
func saveEventLocally(event CalendarEvent) error {
	syncEvent.Lock()
	defer syncEvent.Unlock()

	eventPath := realPath(SysConfig.EventsPath)
	if _, err := os.Stat(eventPath); os.IsNotExist(err) {
		os.MkdirAll(eventPath, 0755)
	}

	e := &LocalEvent{
		Event: event,
	}

	// if event exist, set some properties from the existing event
	existingEvent, err := loadEvent(e.Event.ID + ".json")
	if err == nil {
		e.StartAnnounced = existingEvent.StartAnnounced
		e.CheckStartAnnounced = existingEvent.CheckStartAnnounced
		e.EndAnnounced = existingEvent.EndAnnounced
		e.LastTimeReminded = existingEvent.LastTimeReminded
	}

	eJson, err := json.MarshalIndent(e, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	filePath := path.Join(eventPath, e.Event.ID+".json")
	return os.WriteFile(filePath, eJson, 0644)
}

func loadEvent(name string) (LocalEvent, error) {
	var e LocalEvent

	eventDir := realPath(SysConfig.EventsPath)
	eJson, err := os.ReadFile(path.Join(eventDir, name))
	if err != nil {
		return LocalEvent{}, fmt.Errorf("failed to read event file %s: %v", name, err)
	}
	err = json.Unmarshal(eJson, &e)
	if err != nil {
		return LocalEvent{}, fmt.Errorf("failed to unmarshal event file %s: %v", name, err)
	}

	return e, nil
}

// loadTodayEvents loads today's events from the local storage.
func loadTodayEvents() ([]LocalEvent, error) {
	syncEvent.Lock()
	defer syncEvent.Unlock()

	eventDir := realPath(SysConfig.EventsPath)
	files, err := os.ReadDir(eventDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read events directory: %v", err)
	}

	events := make([]LocalEvent, 0)
	for _, file := range files {
		e, err := loadEvent(file.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to load event %s: %v", file.Name(), err)
		}

		// if event is not scheduled for today, or it's already finished
		if !e.scheduledForToday() || time.Now().After(e.Event.EndTime) {
			continue
		}

		// fix the event time zone
		if e.Event.TimeZone != "" {
			loc, err := time.LoadLocation(e.Event.TimeZone)
			if err != nil {
				logError("failed to load timezone %s for event %s, using local timezone", e.Event.TimeZone, e.Event.ID)
				loc = time.Local
			}
			e.Event.StartTime = e.Event.StartTime.In(loc)
			e.Event.EndTime = e.Event.EndTime.In(loc)
			e.LastTimeReminded = e.LastTimeReminded.In(loc)

			if time.Now().After(e.Event.EndTime) {
				continue // skip events that are already finished
			}
		}

		events = append(events, e)
	}

	return events, nil
}

// syncLocalEvents saves the events to the local storage.
func syncLocalEvents(events []CalendarEvent) error {
	for _, event := range events {
		err := saveEventLocally(event)
		if err != nil {
			return err
		}
	}

	// remove local events that are not in the calendar
	err := removeLocalEventsNotInCalendar(events)
	if err != nil {
		return err
	}

	return nil
}

func removeLocalEventsNotInCalendar(events []CalendarEvent) error {
	syncEvent.Lock()
	defer syncEvent.Unlock()

	eventPath := realPath(SysConfig.EventsPath)
	files, err := os.ReadDir(eventPath)
	if err != nil {
		return fmt.Errorf("failed to read events directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			id := strings.TrimSuffix(file.Name(), ".json")
			found := false
			for _, event := range events {
				if event.ID == id {
					found = true
					break
				}
			}
			if !found {
				err := os.Remove(path.Join(eventPath, file.Name()))
				if err != nil {
					return fmt.Errorf("failed to remove event %s: %v", id, err)
				}
			}
		}
	}
	return nil
}

// setStartAnnounced sets the event start as announced.
func (e *LocalEvent) setStartAnnounced() error {
	e.StartAnnounced = true
	return e.updateEvent()
}

// setStartChecked sets the event start checked.
func (e *LocalEvent) setStartChecked() error {
	e.CheckStartAnnounced = true
	return e.updateEvent()
}

// setEndAnnounced sets the event end as announced.
func (e *LocalEvent) setEndAnnounced() error {
	e.EndAnnounced = true
	return e.updateEvent()
}

// setReminded sets the event as reminded.
func (e *LocalEvent) setReminded() error {
	e.LastTimeReminded = time.Now()
	return e.updateEvent()
}

// updateEvent updates the event's information in the local storage.
func (e *LocalEvent) updateEvent() error {
	syncEvent.Lock()
	defer syncEvent.Unlock()

	eJson, err := json.MarshalIndent(e, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	eventPath := realPath(SysConfig.EventsPath)
	return os.WriteFile(path.Join(eventPath, e.Event.ID+".json"), eJson, 0644)
}

// scheduledForNow returns true if now is within the event's start and end times.
func (e *LocalEvent) scheduledForNow() bool {
	if e.Event.StartTime.IsZero() {
		return false
	}

	now := time.Now()
	if now.After(e.Event.StartTime) && now.Before(e.Event.EndTime) {
		return true
	}

	return false
}

// scheduledNearEnd returns true if now is within a minute of the event's end time.
func (e *LocalEvent) scheduledNearEnd() bool {
	if e.Event.EndTime.IsZero() {
		return false
	}

	now := time.Now()
	if now.After(e.Event.EndTime.Add(-time.Minute)) && now.Before(e.Event.EndTime.Add(time.Minute)) {
		return true
	}

	return false
}

// scheduledForToday returns true if the event is scheduled for today.
func (e *LocalEvent) scheduledForToday() bool {
	// Truncate both times to midnight to compare only the date part
	return e.Event.StartTime.Truncate(24 * time.Hour).Equal(time.Now().Truncate(24 * time.Hour))
}
