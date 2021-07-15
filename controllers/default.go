package controllers

import (
	beego "github.com/beego/beego/v2/server/web"
)

// struct embedding <-- Go mimicing inheritance
type MainController struct {
	beego.Controller
}

func (c *MainController) Get() {
	c.Data["Website"] = "beego.me"
	c.Data["Email"] = "astaxie@gmail.com"
	c.TplName = "index.tpl"
}
