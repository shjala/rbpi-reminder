package main

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"
	"time"
)

var reminding sync.Mutex

func remindCurrentEvents() {
	events, err := loadTodayEvents()
	if err != nil {
		logError("failed to load events: %v", err)
	}

	// speech gen on raspberry pi is slow, so let one round finish before starting another
	reminding.Lock()
	defer reminding.Unlock()

	for _, e := range events {
		if shouldAnnounceEventStart(&e) {
			logDebug("Announcing event start")
			// Set event announced and reminded, otherwise last reminded
			// remains zero and we will remind the task immediately.
			e.setStartAnnounced()
			e.setReminded()
			// Announce it
			text := renderAnnounceStartMessage(&e)
			announceTask(text)
			// if we just announced the start, don't check for reminders
			continue
		}
		if shouldCheckEventStarted(&e) {
			logDebug("Checking event started")
			// Set event start checked
			e.setStartChecked()
			// Announce event start
			text := renderCheckStartMessage(&e)
			announceTask(text)
			// if we checked for start, don't check for other conditions
			continue
		}
		if shouldRemindEvent(&e) {
			logDebug("Reminding event")
			// Set reminded time to now
			e.setReminded()
			// Remind it
			text := renderRemindMessage(&e)
			announceTask(text)
			// if we just reminded, don't check for end
			continue
		}
		if shouldAnnounceEventEnd(&e) {
			logDebug("Announcing event end")
			// Set event end announced
			e.setEndAnnounced()
			// Announce event end
			text := renderAnnounceEndMessage(&e)
			announceTask(text)
		}
	}
}

func shouldAnnounceEventStart(e *LocalEvent) bool {
	if !e.StartAnnounced && e.scheduledForToday() && e.scheduledForNow() {
		return true
	}

	return false
}

func shouldCheckEventStarted(e *LocalEvent) bool {
	if !e.scheduledForToday() || !e.StartAnnounced || e.CheckStartAnnounced {
		return false
	}

	// If we one minute is passed since the event started, check if it has started
	if time.Now().After(e.Event.StartTime.Add(time.Minute)) {
		return true
	}

	return false
}

func shouldAnnounceEventEnd(e *LocalEvent) bool {
	if !e.EndAnnounced && e.scheduledForToday() && e.scheduledNearEnd() {
		return true
	}

	return false
}

func shouldRemindEvent(e *LocalEvent) bool {
	now := time.Now()
	if e.EndAnnounced || e.Event.EndTime.IsZero() || now.Before(e.Event.StartTime) || now.After(e.Event.EndTime) {
		return false
	}

	// check if we are in remiding period, which is every (totalDuration / NotificationRepeats) times
	reminderInterval := e.Event.EndTime.Sub(e.Event.StartTime) / time.Duration(SysConfig.NotificationRepeats)
	if now.After(e.LastTimeReminded.Add(reminderInterval)) {
		return true
	}

	return false
}

func timeLeftString(e *LocalEvent) string {
	left := time.Until(e.Event.EndTime)
	return fmt.Sprintf("%d minutes", int(left.Minutes()))
}

func renderAnnounceStartMessage(e *LocalEvent) string {
	defaultConfig := fmt.Sprintf("Hey! Time to tackle \"%s\"! You have \"%s\" scheduled for now.", e.Event.Description, e.Event.Description)
	tmplText := SysConfig.AnnounceMessageTemplate
	if tmplText == "" {
		tmplText = "Hey! Time to tackle \"{{.Event}}\"! You have \"{{.Event}}\" scheduled for now."
	}

	tmpl, err := template.New("announce_start").Parse(tmplText)
	if err != nil {
		logError("failed to parse announce template: %v", err)
		return defaultConfig
	}

	data := struct {
		Event string
	}{
		Event: e.Event.Description,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		logError("failed to execute announce template: %v", err)
		return defaultConfig
	}

	return buf.String()
}

func renderAnnounceEndMessage(e *LocalEvent) string {
	defaultConfig := fmt.Sprintf("Hey! The \"%s\" is over now!", e.Event.Description)
	tmplText := SysConfig.AnnounceEndMessageTemplate
	if tmplText == "" {
		tmplText = "Hey! The \"{{.Event}}\" is over now!"
	}

	tmpl, err := template.New("announce_end").Parse(tmplText)
	if err != nil {
		logError("failed to parse announce template: %v", err)
		return defaultConfig
	}

	data := struct {
		Event string
	}{
		Event: e.Event.Description,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		logError("failed to execute announce template: %v", err)
		return defaultConfig
	}

	return buf.String()
}

func renderCheckStartMessage(e *LocalEvent) string {
	defaultConfig := fmt.Sprintf("Hey! Did you start \"%s\"?", e.Event.Description)
	tmplText := SysConfig.CheckStartMessageTemplate
	if tmplText == "" {
		tmplText = "Hey! Did you start \"{{.Event}}\"?"
	}

	tmpl, err := template.New("checkstart").Parse(tmplText)
	if err != nil {
		logError("failed to parse checkstart template: %v", err)
		return defaultConfig
	}

	data := struct {
		Event string
	}{
		Event: e.Event.Description,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		logError("failed to execute checkstart template: %v", err)
		return defaultConfig
	}

	return buf.String()
}

func renderRemindMessage(e *LocalEvent) string {
	defaultMessage := fmt.Sprintf("You have %s left for %s", timeLeftString(e), e.Event.Description)
	tmplText := SysConfig.RemindMessageTemplate
	if tmplText == "" {
		tmplText = "You have {{.TimeLeft}} left for {{.Event}}"
	}

	tmpl, err := template.New("remind").Parse(tmplText)
	if err != nil {
		logError("failed to parse remind template: %v", err)
		return defaultMessage
	}

	data := struct {
		Event    string
		TimeLeft string
	}{
		Event:    e.Event.Description,
		TimeLeft: timeLeftString(e),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		logError("failed to execute remind template: %v", err)
		return defaultMessage
	}

	return buf.String()
}

func announceTask(speech string) {
	err := aiSpeak(speech)
	if err != nil {
		logError("failed to announce task: %v", err)
	}
}
