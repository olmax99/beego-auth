package controllers

import (
	"container/list"
	"time"

	"beego-auth/models"
)

type Subscription struct {
	Archive []models.Event      // All the events from the archive.
	New     <-chan models.Event // New events coming in.
}

func newEvent(ep models.EventType, user, msg string) models.Event {
	return models.Event{
		Type:      ep,
		User:      user,
		Timestamp: int(time.Now().Unix()),
		Content:   msg,
	}
}

var (
	publish     = make(chan models.Event, 10)
	waitingList = list.New()
)

// eventqueue Handles all incoming chan messages.
func eventqueue() {
	for {
		select {
		case event := <-publish:
			// Notify waiting list.
			for ch := waitingList.Back(); ch != nil; ch = ch.Prev() {
				ch.Value.(chan bool) <- true
				waitingList.Remove(ch)
			}

			models.NewArchive(event)

			if event.Type == models.EVENT_MESSAGE {
				logB.Info("Message from " + event.User + "; Content: " + event.Content)
			}
		}
	}
}

func init() {
	go eventqueue()
}
