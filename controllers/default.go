package controllers

import (
	"fmt"

	beego "github.com/beego/beego/v2/server/web"
)

// struct embedding <-- Go mimicing inheritance
type MainController struct {
	beego.Controller
}

/////////////////////////////////////////////////////
// NOTE: Deactivate autorender for replacing with  //
// 						   //
// https://github.com/oal/beego-pongo2		   //
/////////////////////////////////////////////////////

// active Content is building the html output from the appropriate templates
func (this *MainController) activeContent(view string) {
	httpport, err := beego.AppConfig.String("httpport")
	if err != nil {
		fmt.Println(err)
	}
	this.Data["Httpport"] = httpport

	this.Layout = "basic-layout.tpl"
	this.LayoutSections = make(map[string]string)
	this.LayoutSections["Header"] = "header.tpl"
	this.LayoutSections["Footer"] = "footer.tpl"
	this.TplName = view + ".tpl"

	// All sessions are created in c.user.Login(): this.SetSession("<name>")
	sess := this.GetSession("auth")
	// if the user is logged in (--> there is a non-nil session),
	// then we set the InSession parameter to a value (any value),
	// which tells the templating engine to use the “Welcome” bar instead of “Login”.
	if sess != nil {
		this.Data["InSession"] = 1 // for login bar in header.tpl
		m := sess.(map[string]interface{})
		// m["first"] refers to the sessios user's first name
		this.Data["First"] = m["first"]
	}
}

func (this *MainController) Get() {
	this.activeContent("index")

	//----------- This page requires login----------------
	sess := this.GetSession("auth")
	if sess == nil {
		this.Redirect("/user/login/home", 302)
		return
	}
	m := sess.(map[string]interface{})
	fmt.Println("INFO [+] Initialize new session")
	fmt.Println("INFO [*] username is", m["username"])
	fmt.Println("INFO [*] logged in at", m["timestamp"])

	err := this.Render()
	if err != nil {
		fmt.Println(err)
	}
}

// Sends the Flash Data 'Notice' content to the main view
func (this *MainController) Notice() {
	this.activeContent("notice")

	// returns current flashData map[string]string and
	// moves the 'flash notice' message into controller's Data output
	flash := beego.ReadFromRequest(&this.Controller)
	if n, ok := flash.Data["notice"]; ok {
		this.Data["notice"] = n
	}

	err := this.Render()
	if err != nil {
		fmt.Println(err)
	}
}
