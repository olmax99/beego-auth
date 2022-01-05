package models

import (
	"container/list"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

type EventType int

const (
	// iota: integer constants autoincrement (keyword)
	EVENT_MESSAGE = iota
)

var (
	logB = logs.NewLogger(10000)
)

type Event struct {
	Type      EventType // MESSAGE
	User      string
	Timestamp int // Unix timestamp (secs)
	Content   string
}

// TEvent transformed event for visualizing
type TEvent struct {
	Type      EventType
	User      string
	Timestamp int    // required for 'lastReceived' logic
	Timeread  string // human readable
	Content   string
}

const archiveSize = 20

// Event archives doubly linked list for stepping through events
var archive = list.New()

// NewArchive saves new event to archive list.
func NewArchive(event Event) {
	if archive.Len() >= archiveSize {
		archive.Remove(archive.Front())
	}
	archive.PushBack(event)
}

// GetEvents returns all events after lastReceived.
func GetEvents(lastReceived int, sessID string) []Event {
	events := make([]Event, 0, archive.Len())
	for event := archive.Front(); event != nil; event = event.Next() {
		e := event.Value.(Event)
		if e.Timestamp > int(lastReceived) && e.User == sessID {
			events = append(events, e)
		}
	}
	return events
}

// TransformEvent Implements data transformation, adds Timeread for message 'show once'
func TransformEvents(obj []Event) []TEvent {
	tevents := make([]TEvent, 0, len(obj))
	for _, v := range obj {
		te := &TEvent{}
		if strings.Contains(v.User, "@") {
			te.User = v.User
		} else {
			te.User = v.User[0:6]
		}
		te.Type = v.Type
		te.Timestamp = v.Timestamp
		te.Timeread = time.Unix(int64(v.Timestamp), 0).String()
		te.Content = v.Content
		tevents = append(tevents, *te)
	}
	return tevents
}
