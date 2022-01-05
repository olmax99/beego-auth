package controllers

import (
	"beego-auth/conf"
	"beego-auth/models"
	"strconv"

	beego "github.com/beego/beego/v2/server/web"
)

// NotifierController handles long polling requests.
type NotifierController struct {
	beego.Controller
}

func (ctl *NotifierController) Prepare() {
	beeC := conf.BeeConf("",
		"sessionname",
	)
	sess := ctl.GetSession(beeC["sessionname"])
	if _, ok := sess.(map[string]interface{}); ok {
		sessID := ctl.CruSession.SessionID(ctl.Ctx.Request.Context())
		logB.Info("[Notifier.Prepare] session found " + sessID)
	}
}

// Fetch method handles fetch archives requests for NotifierController.
func (ctl *NotifierController) Fetch() {
	param := ctl.Ctx.Input.Param(":lastreceived")
	lastReceived, err := strconv.Atoi(param)
	if err != nil {
		logB.Error("Input Param.. " + err.Error())
	}

	sessID := ctl.CruSession.SessionID(ctl.Ctx.Request.Context())
	events := models.GetEvents(int(lastReceived), sessID)
	if len(events) == 1 {
		logB.Info("[Notifier] Fetch().. found '" + strconv.Itoa(len(events)) + "' event.")
		var j []models.TEvent
		j = models.TransformEvents(models.GetEvents(int(lastReceived), sessID))
		ctl.Data["json"] = &j
		if err := ctl.ServeJSON(); err != nil {
			logB.Error("[Notifier] Fetch(): " + err.Error())
		}
		return
	} else if len(events) > 1 {
		logB.Info("[Notifier] Fetch().. found '" + strconv.Itoa(len(events)) + "' events.")
		ctl.Data["json"] = events
		ctl.ServeJSON()
		return
	} else {
		logB.Info("[Notifier] Fetch().. no events found")
		// Wait for new message(s).
		ch := make(chan bool)
		waitingList.PushBack(ch)
		<-ch
	}
}
