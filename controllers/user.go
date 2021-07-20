package controllers

import (
	"beego-auth/models"
	pk "beego-auth/utilities/pbkdf2"

	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego/validation"
	_ "github.com/mattn/go-sqlite3"
	"github.com/twinj/uuid"

	"beego-auth/models"

	beego "github.com/beego/beego/v2/server/web"
)

// TODO separate all beego stuff: no need for testing
func (this *MainController) Login() {
	// ----------------------------GET ---------------------------------------------
	this.activeContent("user/login")

	// 'back' storing the path which was requested before refered here (<-- no session)
	// allow for deeper URL such as l1/l2/l3 represented by l1>l2>l3
	back := strings.Replace(this.Ctx.Input.Param(":back"), ">", "/", -1)
	fmt.Println("INFO [*] ':back' is ..", back)

	// -----------------------------POST --------------------------------------------
	if this.Ctx.Input.Method() == "POST" {
		// Step 1: -----------Validate Form input--------------------------------
		flash := beego.NewFlash()
		email := this.GetString("email")
		password := this.GetString("password")

		valid := validation.Validation{}
		valid.Email(email, "email")
		valid.Required(password, "password")
		if valid.HasErrors() {
			errormap := []string{}
			for _, err := range valid.Errors {
				errormap = append(errormap, "Validation failed on "+err.Key+": "+err.Message+"\n")
			}
			this.Data["Errors"] = errormap
			return
		}
		fmt.Println("INFO [*] Authorization is", email, ":", password)

		// Step 2: -------------Read password hash from database-----------------
		var x pk.PasswordHash

		x.Hash = make([]byte, 32)
		x.Salt = make([]byte, 16)

		o := orm.NewOrm()
		// TODO Get databasename from config
		// ..
		o.Using("default")
		user := models.AuthUser{Email: email}
		err := o.Read(&user, "Email")
		if err == nil {
			// Verify will remove uuid from user, hence if it still exists
			// it indicates that account verification (email) has not been
			// completed
			if user.Reg_key != "" {
				flash.Error("Account not verified")
				flash.Store(&this.Controller)
				return
			}

			// scan in the password hash/salt
			fmt.Println("Password to scan:", user.Password)
			if x.Hash, err = hex.DecodeString(user.Password[:64]); err != nil {
				fmt.Println("ERROR [-] ..", err)
			}
			if x.Salt, err = hex.DecodeString(user.Password[64:]); err != nil {
				fmt.Println("ERROR [-] ..", err)
			}
			fmt.Println("INFO [*] decoded password is", x)
		} else {
			flash.Error("No such user/email")
			flash.Store(&this.Controller)
			return
		}

		// Step 3: ------------- Compare submitted password with database--------
		if !pk.MatchPassword(password, &x) {
			flash.Error("Bad password")
			flash.Store(&this.Controller)
			return
		}

		// Step 4: ------------ Create session and go back to previous page------
		m := make(map[string]interface{})
		m["first"] = user.First
		m["username"] = email
		m["timestamp"] = time.Now()
		this.SetSession("auth", m)
		this.Redirect("/"+back, 302)
	}
	err := this.Render()
	if err != nil {
		fmt.Println(err)
	}
}

func (this *MainController) Logout() {
	this.activeContent("logout")
	this.DelSession("auth")
	this.Redirect("/home", 302)

	err := this.Render()
	if err != nil {
		fmt.Println(err)
	}
}

func (this *MainController) Register() {
	this.activeContent("user/register")

	if this.Ctx.Input.Method() == "POST" {
		// Step 1: -------------- Validate Form input--------------------------
		flash := beego.NewFlash()

		first := this.GetString("first")
		last := this.GetString("last")
		email := this.GetString("email")
		password := this.GetString("password")
		password2 := this.GetString("password2")

		valid := validation.Validation{}
		valid.Required(first, "first")
		valid.Email(email, "email")
		valid.MinSize(password, 6, "password")
		valid.Required(password2, "password2")
		if valid.HasErrors() {
			errormap := []string{}
			for _, err := range valid.Errors {
				errormap = append(errormap, "Validation failed on "+err.Key+": "+err.Message+"\n")
			}
			this.Data["Errors"] = errormap
			return
		}
		if password != password2 {
			flash.Error("Passwords don't match")
			flash.Store(&this.Controller)
			return
		}
		h := pk.HashPassword(password)

		// Step 3: -------------- Save user info to database-------------------
		o := orm.NewOrm()
		o.Using("default")

		user := models.AuthUser{First: first, Last: last, Email: email}

		// Convert password hash to string
		user.Password = hex.EncodeToString(h.Hash) + hex.EncodeToString(h.Salt)

		// Add user to database with new uuid and
		u := uuid.NewV4()
		user.Reg_key = u.String()
		_, err := o.Insert(&user)
		if err != nil {
			// TODO confirm if other errors need to be handled??
			flash.Error(email + " already registered")
			flash.Store(&this.Controller)
			return
		}

		// TODO sendgrid binding
		// Step 4: --------------- Send verification email---------------------
		// if !sendVerification(email, u.String()) {
		// 	flash.Error("Unable to send verification email")
		// 	flash.Store(&this.Controller)
		// 	return
		// }

		// Step 5: --------------- Append confirmation to flash & redirect-----
		flash.Notice("Your account has been created. You must verify the account in your email.")
		flash.Store(&this.Controller)
		this.Redirect("/notice", 302)
	}

	err := this.Render()
	if err != nil {
		fmt.Println(err)
	}
}

// Set verification attribute in current session context
func (this *MainController) Verify() {
	this.activeContent("user/verify")

	u := this.Ctx.Input.Param(":uuid")
	o := orm.NewOrm()
	o.Using("default")

	// Get user from data base by filtering on uuid
	user := models.AuthUser{Reg_key: u}
	err := o.Read(&user, "Reg_key")
	if err == nil {
		this.Data["Verified"] = 1
		// Remove registration key after context has 'Verified=1'
		user.Reg_key = ""
		if _, err := o.Update(&user); err != nil {
			delete(this.Data, "Verified")
		}
	}

	errR := this.Render()
	if errR != nil {
		fmt.Println(err)
	}
}

func (this *MainController) Profile() {
	this.activeContent("user/profile")

	//******** This page requires login
	sess := this.GetSession("acme")
	if sess == nil {
		this.Redirect("/user/login/home", 302)
		return
	}
	m := sess.(map[string]interface{})

	flash := beego.NewFlash()

	//******** Read password hash from database
	var x pk.PasswordHash

	x.Hash = make([]byte, 32)
	x.Salt = make([]byte, 16)

	o := orm.NewOrm()
	o.Using("default")
	user := models.AuthUser{Email: m["username"].(string)}
	err := o.Read(&user, "Email")
	if err == nil {
		// scan in the password hash/salt
		if x.Hash, err = hex.DecodeString(user.Password[:64]); err != nil {
			fmt.Println("ERROR:", err)
		}
		if x.Salt, err = hex.DecodeString(user.Password[64:]); err != nil {
			fmt.Println("ERROR:", err)
		}
	} else {
		flash.Error("Internal error")
		flash.Store(&this.Controller)
		return
	}

	// this deferred function ensures that the correct fields from the database are displayed
	defer func(this *MainController, user *models.AuthUser) {
		this.Data["First"] = user.First
		this.Data["Last"] = user.Last
		this.Data["Email"] = user.Email
	}(this, &user)

	if this.Ctx.Input.Method() == "POST" {
		first := this.GetString("first")
		last := this.GetString("last")
		email := this.GetString("email")
		current := this.GetString("current")
		password := this.GetString("password")
		password2 := this.GetString("password2")
		valid := validation.Validation{}
		valid.Required(first, "first")
		valid.Email(email, "email")
		valid.Required(current, "current")
		if valid.HasErrors() {
			errormap := []string{}
			for _, err := range valid.Errors {
				errormap = append(errormap, "Validation failed on "+err.Key+": "+err.Message+"\n")
			}
			this.Data["Errors"] = errormap
			return
		}

		if password != "" {
			valid.MinSize(password, 6, "password")
			valid.Required(password2, "password2")
			if valid.HasErrors() {
				errormap := []string{}
				for _, err := range valid.Errors {
					errormap = append(errormap, "Validation failed on "+err.Key+": "+err.Message+"\n")
				}
				this.Data["Errors"] = errormap
				return
			}

			if password != password2 {
				flash.Error("Passwords don't match")
				flash.Store(&this.Controller)
				return
			}
			h := pk.HashPassword(password)

			// Convert password hash to string
			user.Password = hex.EncodeToString(h.Hash) + hex.EncodeToString(h.Salt)
		}

		//******** Compare submitted password with database
		if !pk.MatchPassword(current, &x) {
			flash.Error("Bad current password")
			flash.Store(&this.Controller)
			return
		}

		//******** Save user info to database
		user.First = first
		user.Last = last
		user.Email = email

		_, err := o.Update(&user)
		if err == nil {
			flash.Notice("Profile updated")
			flash.Store(&this.Controller)
			m["username"] = email
		} else {
			flash.Error("Internal error")
			flash.Store(&this.Controller)
			return
		}
	}

	errR := this.Render()
	if err != nil {
		fmt.Println(errR)
	}
}

func (this *MainController) Remove() {
	this.activeContent("user/remove")

	//******** This page requires login
	sess := this.GetSession("acme")
	if sess == nil {
		this.Redirect("/user/login/home", 302)
		return
	}
	m := sess.(map[string]interface{})

	if this.Ctx.Input.Method() == "POST" {
		current := this.GetString("current")
		valid := validation.Validation{}
		valid.Required(current, "current")
		if valid.HasErrors() {
			errormap := []string{}
			for _, err := range valid.Errors {
				errormap = append(errormap, "Validation failed on "+err.Key+": "+err.Message+"\n")
			}
			this.Data["Errors"] = errormap
			return
		}

		flash := beego.NewFlash()

		//******** Read password hash from database
		var x pk.PasswordHash

		x.Hash = make([]byte, 32)
		x.Salt = make([]byte, 16)

		o := orm.NewOrm()
		o.Using("default")
		user := models.AuthUser{Email: m["username"].(string)}
		err := o.Read(&user, "Email")
		if err == nil {
			// scan in the password hash/salt
			if x.Hash, err = hex.DecodeString(user.Password[:64]); err != nil {
				fmt.Println("ERROR:", err)
			}
			if x.Salt, err = hex.DecodeString(user.Password[64:]); err != nil {
				fmt.Println("ERROR:", err)
			}
		} else {
			flash.Error("Internal error")
			flash.Store(&this.Controller)
			return
		}

		//******** Compare submitted password with database
		if !pk.MatchPassword(current, &x) {
			flash.Error("Bad current password")
			flash.Store(&this.Controller)
			return
		}

		//******** Delete user record
		_, err = o.Delete(&user)
		if err == nil {
			flash.Notice("Your account is deleted.")
			flash.Store(&this.Controller)
			this.DelSession("acme")
			this.Redirect("/notice", 302)
		} else {
			flash.Error("Internal error")
			flash.Store(&this.Controller)
			return
		}
	}

	err := this.Render()
	if err != nil {
		fmt.Println(err)
	}
}
