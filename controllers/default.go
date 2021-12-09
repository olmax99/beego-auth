package controllers

import (
	"beego-auth/conf"
	"fmt"

	beego "github.com/beego/beego/v2/server/web"
)

// struct embedding <-- Go mimicing inheritance
type DefaultController struct {
	beego.Controller
}

/////////////////////////////////////////////////////
// TODO: Deactivate autorender for replacing with  //
// 						   //
// https://github.com/oal/beego-pongo2		   //
/////////////////////////////////////////////////////

// active Content is building the html output from the appropriate templates
func (ctl *DefaultController) activeContent(view string) {
	beeC := conf.BeeConf("",
		"httpport",
		"sessionname",
	)
	ctl.Data["Httpport"] = beeC["httpport"]

	ctl.Layout = "basic-layout.tpl"
	ctl.LayoutSections = make(map[string]string)
	ctl.LayoutSections["Header"] = "header.tpl"
	ctl.LayoutSections["Footer"] = "footer.tpl"
	ctl.TplName = view + ".tpl"

	// All sessions are created in c.user.Login(): ctl.SetSession("<name>")
	if view != "/user/login/console" {
		sess := ctl.GetSession(beeC["sessionname"])
		// if the user is logged in (--> there is a non-nil session),
		// then we set the InSession parameter to a value (any value),
		// which tells the templating engine to use the “Welcome” bar instead of “Login”.
		if sess != nil {
			ctl.Data["InSession"] = 1 // for login bar in header.tpl
			m := sess.(map[string]interface{})
			// m["first"] refers to the sessions user's first name
			ctl.Data["First"] = m["first"]
		}
	}
}

func (ctl *DefaultController) Get() {
	ctl.activeContent("index")

	beeC := conf.BeeConf("",
		"sessionname",
	)

	sess := ctl.GetSession(beeC["sessionname"])
	m := sess.(map[string]interface{})
	fmt.Println("INFO [+] Initialize new session")
	fmt.Println("INFO [*] username is", m["username"])
	fmt.Println("INFO [*] logged in at", m["timestamp"])

	err := ctl.Render()
	if err != nil {
		fmt.Println(err)
	}
}

// TODO: Probably an Api endpoint or callback handler for JS
// Sends the Flash Data 'Notice' content to the main view
func (ctl *DefaultController) Notice() {
	ctl.activeContent("notice")

	// returns current flashData map[string]string and
	// moves the 'flash notice' message into controller's Data output
	flash := beego.ReadFromRequest(&ctl.Controller)
	if n, ok := flash.Data["notice"]; ok {
		ctl.Data["notice"] = n
	}

	err := ctl.Render()
	if err != nil {
		fmt.Println(err)
	}
}
