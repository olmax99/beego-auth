package controllers

import (
	"fmt"

	beego "github.com/beego/beego/v2/server/web"
)

// struct embedding <-- Go mimicing inheritance
type MainController struct {
	beego.Controller
}

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

	// 'auth' refers to the projects name
	// TODO How are sessions be named
	sess := this.GetSession("auth")

	// TODO Explain Data.InSession logic
	// What does m["first"] refer to??
	if sess != nil {
		this.Data["InSession"] = 1 // for login bar in header.tpl
		m := sess.(map[string]interface{})
		this.Data["First"] = m["first"]
	}
}

func (this *MainController) Get() {
	this.activeContent("index")
}
