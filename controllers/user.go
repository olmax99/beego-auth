package controllers

import (
	_ "github.com/mattn/go-sqlite3"

	beego "github.com/beego/beego/v2/server/web"
)

type UserController struct {
	beego.Controller
}

// @Title CreateOncUser
// @Description create users
// @Success 200 {int} models.NewOncUser.clientId
// @route /create [get]
// func (u *UserController) Get() {
// 	// Requires 'SessionOn = true', 'SessionProvider = "memory"' (default)
// 	// v := c.GetSession("asta")
// 	// if v == nil {
// 	// 	c.SetSession("asta", int(1))
// 	// 	c.Data["num"] = 0
// 	// } else {
// 	// 	c.SetSession("asta", v.(int)+1)
// 	// 	c.Data["num"] = v.(int)
// 	// }
// 	res := models.AddUser()
// 	u.Data["json"] = map[string]int64{"clientId": res}
// 	u.ServeJSON()
// }
