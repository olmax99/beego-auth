package controllers

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"beego-auth/conf"
	"beego-auth/models"

	mail "beego-auth/pkg/sendgridv1"
	crypt "beego-auth/pkg/vaultv1"

	"github.com/beego/beego/v2/adapter/orm"
	"github.com/beego/beego/v2/adapter/utils"
	"github.com/beego/beego/v2/core/validation"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/twinj/uuid"

	beego "github.com/beego/beego/v2/server/web"
)

// TODO separate all beego stuff: no need for testing
func (ctl *DefaultController) Login() {
	// ----------------------------GET ---------------------------------------------
	ctl.activeContent("user/login")
	// 'back' storing the path which was requested before refered here (<-- no session)
	// allow for deeper URL such as l1/l2/l3 represented by l1>l2>l3
	back := strings.ReplaceAll(ctl.Ctx.Input.Param(":back"), ">", "/")
	fmt.Println("INFO [*] ':back' is ..", back)

	// -----------------------------POST --------------------------------------------
	if ctl.Ctx.Input.Method() == "POST" {

		flash := beego.NewFlash()

		beeC := conf.BeeConf("",
			"sessionname",
			"db::beego_db_alias",
			"vault::beego_vault_transit_key",
			"vault::beego_vault_address",
			"vault::beego_vault_token",
		)

		// Step 1: -----------Validate Form input--------------------------------
		email := ctl.GetString("email")
		password := ctl.GetString("password")

		valid := validation.Validation{}
		valid.Email(email, "email")
		valid.Required(password, "password")
		if valid.HasErrors() {
			errormap := []string{}
			for _, err := range valid.Errors {
				errormap = append(errormap, "Validation failed on "+err.Key+": "+err.Message+"\n")
			}
			ctl.Data["Errors"] = errormap
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				fmt.Printf("ERROR [*] Login.Validation().. %s", err)
			}
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
			return
		}
		// Verify() will remove uuid from user, hence if it still exists
		// it indicates that account verification (email) has not been
		// completed
		user := ctl.ReadAuthUser(email, beeC["beego_db_alias"])()

		if user.Reg_key != "" || user.Active != "" {
			flash.Error("Account not active.")
			flash.Store(&ctl.Controller)
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}
		// Step 2: ------ Compare password with db------------------------
		// TODO: Try handle dependency injection with google/wire (user -> crypt)
		crypt := crypt.NewCrypter(beeC)
		if err := crypt.SetVaultv1(user.Password); err != nil {
			log.Printf("ERROR [*] VaultDecryptVal.Set().. %v", err)
		}
		if !crypt.Match(password) {
			flash.Error("Bad password")
			flash.Store(&ctl.Controller)
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}
		// Step 3: ------ Create session and go back to previous page------
		m := make(map[string]interface{})
		m["first"] = user.First
		m["username"] = email
		m["timestamp"] = time.Now()
		ctl.SetSession(beeC["sessionname"], m)
		ctl.Redirect("/"+back, 302)
	}
	errR := ctl.Render()
	if errR != nil {
		fmt.Println(errR)
	}
}

func (ctl *DefaultController) Logout() {
	flash := beego.NewFlash()

	beeC := conf.BeeConf("",
		"sessionname",
	)

	ctl.activeContent("user/logout")
	ctl.DelSession(beeC["sessionname"])

	flash.Notice("Thanks for checking in, Bye.")
	flash.Store(&ctl.Controller)
	ctl.Redirect("/notice", 302)
}

func (ctl *DefaultController) Register() {
	// ----------------------------GET ---------------------------------------------
	ctl.activeContent("user/register")

	// -----------------------------POST -------------------------------------------
	if ctl.Ctx.Input.Method() == "POST" {
		flash := beego.NewFlash()

		beeC := conf.BeeConf("",
			"httpport",
			"db::beego_db_alias",
			"vault::beego_vault_address",
			"vault::beego_vault_token",
			"vault::beego_vault_transit_key",
			"sendgrid::beego_sg_own_support",
			"sendgrid::beego_sg_api_key",
		)

		first := ctl.GetString("first")
		last := ctl.GetString("last")
		email := ctl.GetString("email")
		password := ctl.GetString("password")
		password2 := ctl.GetString("password2")

		// Step 1: -------------- Validate Form input---------------------------
		// TODO: implement https://gowalker.org/github.com/beego/beego/v2/core/validation#ValidFormer:
		// - Adjust user struct with valid tags
		valid := validation.Validation{}
		valid.Required(first, "first")
		valid.Email(email, "email")
		valid.MinSize(password, 12, "password")
		valid.Required(password2, "password2")
		if valid.HasErrors() {
			// return all recorded errors at once
			errormap := []string{}
			for _, err := range valid.Errors {
				errormap = append(errormap, "Validation failed on "+err.Key+": "+err.Message+"\n")
			}
			ctl.Data["Errors"] = errormap
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				fmt.Printf("ERROR [*] Login.Validation().. %s", err)
			}
			errR := ctl.Render()
			if errR != nil {
				fmt.Printf("ERROR [*] DefaultController.Register() validation.. %v", errR)
			}
		}
		if password != password2 {
			flash.Error("Passwords don't match")
			flash.Store(&ctl.Controller)
			return
		}
		// Step 5: -------------- Save user info to database--------------------
		u := uuid.NewV4() // new user verify uuid

		crypt := crypt.NewCrypter(beeC)
		if err := crypt.SetValue(password2); err != nil {
			log.Printf("ERROR [*] DefaultController.Register(), VaultCrypter.en().. %v", err)
		}

		o := orm.NewOrm()
		o.Using(beeC["beego_db_alias"])
		user := new(models.AuthUser)
		user.First = first
		user.Last = last
		user.Email = email
		user.Password = crypt.GetVaultv1()
		user.Reg_key = u.String()

		_, err := o.Insert(user)
		if err != nil {
			log.Printf("ERROR [*] Register() Insert.. %v", err)
			// TODO: confirm if other errors need to be handled??
			flash.Error(email + " already exists.")
			flash.Store(&ctl.Controller)
			errR := ctl.Render()
			if errR != nil {
				fmt.Printf("ERROR [*] DefaultController.Register() save user.. %v", errR)
			}

		}

		// Step 6: --------------- Send verification email----------------------
		// TODO: Verify successfully send (webhook??)
		if !mail.SendVerification(user, u.String(), beeC) {
			flash.Error("Unable to send verification email.")
			flash.Store(&ctl.Controller)
			return
		}

		// Step 7: --------------- Append confirmation to flash & redirect------
		flash.Notice("Your account has been created. You must verify the account in your email.")
		flash.Store(&ctl.Controller)
		ctl.Redirect("/notice", 302)
	}

	// explicit render (can be omitted by setting: 'autorender = true')
	err := ctl.Render()
	if err != nil {
		fmt.Println(err)
	}
}

func (ctl *DefaultController) Verify() {
	// ----------------------------- GET--------------------------------------------
	ctl.activeContent("user/verify")

	beeC := conf.BeeConf("",
		"db::beego_db_alias",
	)

	u := ctl.Ctx.Input.Param(":uuid")
	o := orm.NewOrm()
	o.Using(beeC["beego_db_alias"])

	// Get user from data base by filtering on uuid
	user := &models.AuthUser{Reg_key: u}
	err := o.Read(user, "Reg_key")
	if err == nil {
		ctl.Data["Verified"] = 1
		// Remove registration key after context has 'Verified=1'
		user.Reg_key = ""
		if _, err := o.Update(user); err != nil {
			delete(ctl.Data, "Verified")
		}
	}

	// explicit render (can be omitted by setting: 'autorender = true')
	errR := ctl.Render()
	if errR != nil {
		fmt.Println(errR)
	}
}

func (ctl *DefaultController) Cancel() {
	// ----------------------------- GET--------------------------------------------
	ctl.activeContent("user/cancel")

	flash := beego.NewFlash()

	beeC := conf.BeeConf("",
		"db::beego_db_alias",
	)

	u := ctl.Ctx.Input.Param(":uuid")

	o := orm.NewOrm()
	o.Using(beeC["beego_db_alias"])
	user := &models.AuthUser{Active: u}
	err := o.Read(user, "Active")
	if err == nil && user.Active == u {
		ctl.Data["Cancelled"] = 1
		user.Clc_date = time.Now().UTC()
		if _, err := o.Update(user); err != nil {
			log.Printf("Error [*] Cancel failed.. %v", err)
			flash.Error("Deactivation failed. Please try again..")
			flash.Store(&ctl.Controller)
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}
		delete(ctl.Data, "Cancelled")
		ctl.DestroySession()
		errR := ctl.Render()
		if errR != nil {
			fmt.Println(errR)
		}
	}
	flash.Error("Deactivation failed. Please try again..")
	flash.Store(&ctl.Controller)
	errR := ctl.Render()
	if errR != nil {
		fmt.Println(errR)
	}
}

func (ctl *DefaultController) Genpass1() {
	// ----------------------------- GET--------------------------------------------
	// TODO: Improve by both reading uuid and allow user to post his current password
	// At this point it was not clear on how to combine those two
	flash := beego.NewFlash()

	beeC := conf.BeeConf("",
		"sessionname",
		"db::beego_db_alias",
	)
	u := ctl.Ctx.Input.Param(":uuid")

	o := orm.NewOrm()
	o.Using(beeC["beego_db_alias"])
	user := &models.AuthUser{Reset: u}
	err := o.Read(user, "Reset")
	if err == nil || user.Reset == u {
		// set the 24h limit for next passwd reset
		user.Rst_date = time.Now().UTC()
		if _, err := o.Update(user); err != nil {
			log.Printf("Error [*] Genpass1 Rst_date.. %v", err)
			flash.Notice("You can now login with your temporary password.")
			flash.Store(&ctl.Controller)
			ctl.Redirect("/notice", 302)
		}
		// Step 2: ----------- Create new session---------------------------
		m := make(map[string]interface{})
		m["first"] = user.First
		m["username"] = user.Email
		m["timestamp"] = time.Now()
		ctl.SetSession(beeC["sessionname"], m)
		ctl.Redirect("/user/genpass2", 307)
	}
	flash.Notice("You can now login with your temporary password.")
	flash.Store(&ctl.Controller)
	ctl.Redirect("/notice", 302)
}

func (ctl *DefaultController) Genpass2() {
	ctl.activeContent("user/genpass2")

	flash := beego.NewFlash()

	beeC := conf.BeeConf("",
		"sessionname",
		"db::beego_db_alias",
		"vault::beego_vault_address",
		"vault::beego_vault_token",
		"vault::beego_vault_transit_key",
	)

	sess := ctl.GetSession(beeC["sessionname"])
	m := sess.(map[string]interface{})

	// Step 1:---------- Read current password hash from database-----------------
	user := ctl.ReadAuthUser(m["username"].(string), beeC["beego_db_alias"])()
	crypt := crypt.NewCrypter(beeC)
	crypt.SetVaultv1(user.Password)

	if ctl.Ctx.Input.Method() == "POST" {
		current := ctl.GetString("current")
		password := ctl.GetString("password")
		password2 := ctl.GetString("password2")
		valid := validation.Validation{}
		valid.MinSize(password, 12, "password")
		valid.Required(password2, "password2")
		valid.Required(current, "current")
		if valid.HasErrors() {
			// return all recorded errors at once
			errormap := []string{}
			for _, err := range valid.Errors {
				errormap = append(errormap, "Validation failed on "+err.Key+": "+err.Message+"\n")
			}
			ctl.Data["Errors"] = errormap
			errR := ctl.Render()
			if errR != nil {
				fmt.Printf("ERROR [*] DefaultController.Register() validation.. %v", errR)
			}
		}
		if password != password2 {
			flash.Error("Passwords don't match.")
			flash.Store(&ctl.Controller)
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}
		if crypt.GetValue() != current {
			flash.Error("The current password is incorrect.")
			flash.Store(&ctl.Controller)
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}
		// Step 3: ---------- Update Password in database----------------------

		if err := crypt.SetVaultv1(password); err != nil {
			log.Printf("ERROR [*] Genpass2 VaultDecrypter.set().. %v", err)
		}
		user.Password = crypt.GetVaultv1()
		o := orm.NewOrm()
		o.Using(beeC["beego_db_alias"])
		_, err := o.Update(user)
		if err != nil {
			flash.Error("Password reset failed. Please try again..")
			flash.Store(&ctl.Controller)
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}
		flash.Notice("Your temporary password has been updated.")
		flash.Store(&ctl.Controller)
		ctl.Redirect("/notice", 302)
	}
	errR := ctl.Render()
	if errR != nil {
		fmt.Println(errR)
	}
}

func (ctl *DefaultController) Profile() {
	// ----------------------------- GET--------------------------------------------
	ctl.activeContent("user/profile")

	flash := beego.NewFlash()

	beeC := conf.BeeConf("",
		"sessionname",
		"db::beego_db_alias",
		"vault::beego_vault_address",
		"vault::beego_vault_token",
		"vault::beego_vault_transit_key",
	)

	sess := ctl.GetSession(beeC["sessionname"])
	m := sess.(map[string]interface{})

	// Step 1:---------- Read current password hash from database-----------------
	crypt := crypt.NewCrypter(beeC)
	user := ctl.ReadAuthUser(m["username"].(string), beeC["beego_db_alias"])()
	if user.Reg_key != "" {
		flash.Error("Account not verified")
		flash.Store(&ctl.Controller)
		return
	}
	if err := crypt.SetVaultv1(user.Password); err != nil {
		log.Printf("ERROR [*] Profile() VaultDecrypter.set().. %v", err)
	}
	ctl.Data["First"] = user.First
	ctl.Data["Last"] = user.Last
	ctl.Data["Email"] = user.Email

	// deferred func ensures that the correct fields from the database are displayed
	defer func(ctl *DefaultController, user *models.AuthUser) {
		ctl.Data["First"] = user.First
		ctl.Data["Last"] = user.Last
		ctl.Data["Email"] = user.Email
	}(ctl, user)

	// ----------------------------- POST-------------------------------------------
	// Profile can be used to update user profile data
	// Profile is providing change user password functionality
	if ctl.Ctx.Input.Method() == "POST" {
		first := ctl.GetString("first")
		last := ctl.GetString("last")
		email := ctl.GetString("email")
		current := ctl.GetString("current")
		valid := validation.Validation{}
		valid.Required(first, "first")
		valid.Email(email, "email")
		valid.Required(current, "current")
		if valid.HasErrors() {
			errormap := []string{}
			for _, err := range valid.Errors {
				errormap = append(errormap, "Validation failed on "+err.Key+": "+err.Message+"\n")
			}
			ctl.Data["Errors"] = errormap
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}
		if email != ctl.Data["Email"] {
			flash.Error("Your Email cannot be changed. You must create a new account.")
			flash.Store(&ctl.Controller)
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}

		// Step 2:---------- Compare submitted password with database---------
		// Ensure that controller drops out if current password does not match
		// with DB
		if !crypt.Match(current) {
			flash.Error("Bad current password")
			flash.Store(&ctl.Controller)
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}

		// Step 3: --------- update user info in db---------------------------
		user.First = first
		user.Last = last

		o := orm.NewOrm()
		o.Using(beeC["beego_db_alias"])
		_, err := o.Update(user)
		if err == nil {
			flash.Notice("Profile updated")
			flash.Store(&ctl.Controller)
			m["username"] = email
		} else {
			flash.Error("Internal error")
			flash.Store(&ctl.Controller)
			return
		}
	}

	// explicit render (can be omitted by setting: 'autorender = true')
	errR := ctl.Render()
	if errR != nil {
		fmt.Println(errR)
	}
}

func (ctl *DefaultController) Remove() {
	// ----------------------------- GET------------------------------------------
	ctl.activeContent("user/remove")

	beeC := conf.BeeConf("",
		"httpport",
		"sessionname",
		"db::beego_db_alias",
		"vault::beego_vault_address",
		"vault::beego_vault_token",
		"vault::beego_vault_transit_key",
		"sendgrid::beego_sg_own_support",
		"sendgrid::beego_sg_api_key",
	)

	sess := ctl.GetSession(beeC["sessionname"])
	m := sess.(map[string]interface{})

	// -----------------------------POST -----------------------------------------
	if ctl.Ctx.Input.Method() == "POST" {
		current := ctl.GetString("current")
		valid := validation.Validation{}
		valid.Required(current, "current")
		if valid.HasErrors() {
			errormap := []string{}
			for _, err := range valid.Errors {
				errormap = append(errormap, "Validation failed on "+err.Key+": "+err.Message+"\n")
			}
			ctl.Data["Errors"] = errormap
			return
		}

		flash := beego.NewFlash()

		// Step 1---------- Read password hash from database-------------
		crypt := crypt.NewCrypter(beeC)
		user := ctl.ReadAuthUser(m["username"].(string), beeC["beego_db_alias"])()
		// Verify() will remove uuid from user, hence if it still exists
		// it indicates that account verification (email) has not been
		// completed
		if user.Reg_key != "" {
			flash.Error("Account not verified")
			flash.Store(&ctl.Controller)
			return
		}
		if err := crypt.SetVaultv1(user.Password); err != nil {
			log.Printf("ERROR [*] Remove() VaultDecryptVal.Set().. %v", err)
		}

		// User is required to provide password in order to proceed
		// Step 2: -------- Compare submitted password with database------
		if !crypt.Match(current) {
			flash.Error("Bad current password")
			flash.Store(&ctl.Controller)
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}

		o := orm.NewOrm()
		o.Using(beeC["beego_db_alias"])
		a := uuid.NewV4()
		user.Active = a.String()
		_, err := o.Update(user)
		if err != nil {
			flash.Error("Deactivation failed. Please try again..")
			flash.Store(&ctl.Controller)
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}

		// Step 3:--------- Delete user record----------------------------
		// Send email
		if !mail.SendCancellation(user, a.String(), beeC) {
			flash.Error("Unable to send cancellation email")
			flash.Store(&ctl.Controller)
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}

		// Step 7: --------------- Append confirmation to flash & redirect------
		flash.Notice("We are sad to see you go. You will receive an email soon with instructions on how to cancel your account.")
		flash.Store(&ctl.Controller)
		ctl.Redirect("/notice", 302)
	}

	// explicit render (can be omitted by setting: 'autorender = true')
	err := ctl.Render()
	if err != nil {
		fmt.Println(err)
	}
}

func (ctl *DefaultController) Reset() {
	ctl.activeContent("user/reset")

	beeC := conf.BeeConf("",
		"httpport",
		"sessionname",
		"db::beego_db_alias",
		"vault::beego_vault_address",
		"vault::beego_vault_token",
		"vault::beego_vault_transit_key",
		"sendgrid::beego_sg_own_support",
		"sendgrid::beego_sg_api_key",
	)

	// -----------------------------POST -----------------------------------------
	if ctl.Ctx.Input.Method() == "POST" {
		current := ctl.GetString("current")
		valid := validation.Validation{}
		valid.Email(current, "current")
		valid.Required(current, "current")
		if valid.HasErrors() {
			errormap := []string{}
			for _, err := range valid.Errors {
				errormap = append(errormap, "Validation failed on "+err.Key+": "+err.Message+"\n")
			}
			ctl.Data["Errors"] = errormap
			errR := ctl.Render()
			if errR != nil {
				fmt.Printf("ERROR [*] DefaultController.Reset() validation.. %v", errR)
			}
		}

		flash := beego.NewFlash()

		// Step 1---------- Read password hash from database-------------
		user := ctl.ReadAuthUser(current, beeC["beego_db_alias"])()
		// Drop out if last password reset was within 24h
		// The first time an empty time field is accepted
		if !user.Rst_date.Equal(time.Time{}) {
			tn := time.Now().UTC()
			tr := user.Rst_date
			if 24*time.Hour > tn.Sub(tr) {
				flash.Error("There must be at least 24h after your last reset.")
				flash.Store(&ctl.Controller)
				errR := ctl.Render()
				if errR != nil {
					fmt.Println(errR)
				}
			}
		}
		// Step 2: ------ Generate New Random Password----------------
		r := uuid.NewV4()
		newPass := utils.RandomCreateBytes(16)
		crypt := crypt.NewCrypter(beeC)
		if err := crypt.FromBytes(newPass); err != nil {
			log.Printf("ERROR [*] DefaultController.Register(), VaultCrypter.En().. %v", err)
		}
		user.Reset = r.String()
		user.Password = crypt.GetVaultv1()
		o := orm.NewOrm()
		o.Using(beeC["beego_db_alias"])
		_, err := o.Update(user)
		if err != nil {
			flash.Error("Deactivation failed. Please try again..")
			flash.Store(&ctl.Controller)
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}

		// Step 3: ------ Send Reset Email-----------------------------
		if !mail.SendReset(user, r.String(), newPass, beeC) {
			flash.Error("Unable to send password reset email")
			flash.Store(&ctl.Controller)
			errR := ctl.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}

		// Step 4: ------ Append confirmation to flash & redirect------
		flash.Notice("Your password has been reset. Please follow the instructions in your confirmation email.")
		flash.Store(&ctl.Controller)
		ctl.Redirect("/notice", 302)
	}

	// explicit render (can be omitted by setting: 'autorender = true')
	errR := ctl.Render()
	if errR != nil {
		fmt.Println(errR)
	}
}

// TODO move to model - needs separation from controller
// TODO: Error cases need to close request context!!
// ReadAuthUser ensures that subsequent actions do have an existing user struct pointer
// and if not differentiates the Read errors
func (c *DefaultController) ReadAuthUser(se, al string) func() *models.AuthUser {
	flash := beego.NewFlash()
	o := orm.NewOrm()
	o.Using(al)
	user := &models.AuthUser{Email: se}
	fc := func() *models.AuthUser {
		err := o.Read(user, "Email")
		if err != nil {
			switch errM := err.Error(); {
			case errM == orm.ErrNoRows.Error():
				log.Printf("ERROR [*] No result found.. %v", err)
				flash.Error("No such user/email")
				flash.Store(&c.Controller)
				errR := c.Render()
				if errR != nil {
					fmt.Println(errR)
				}
			case errM == orm.ErrMissPK.Error():
				log.Printf("ERROR [*] No primary key found.. %v", err)
				flash.Error("No such user/email")
				flash.Store(&c.Controller)
				errR := c.Render()
				if errR != nil {
					fmt.Println(errR)
				}
			default:
				log.Printf("ERROR [*] Something else went wrong.. %v", err)
				flash.Error("No such user/email")
				flash.Store(&c.Controller)
				errR := c.Render()
				if errR != nil {
					fmt.Println(errR)
				}
			}
		}
		return user
	}
	return fc
}

// TODO mode to model, create non-Beego related user tasks (combine with ReadAuthUser)
func doStuffWithAuthUser(u *models.AuthUser) error {
	log.Printf("DEBUG [*] do something with user: %#v", u)
	return nil
}

// customize filters for fine grain authorization
var FilterUser = func(ctx *context.Context) {
	beeC := conf.BeeConf("",
		"sessionname",
	)
	// Do not authorize when:
	// 1. a session does not exists
	// 2. the request does not come from Request.RequestURI "/login"
	// 3. the request does not come from Request.RequestURI "/register"
	rawurl, err := url.Parse(ctx.Input.Referer())
	if err != nil {
		log.Printf("DEBUG [*] FilterUser ctx.Input.Referer: %s", err)
	}
	_, ok := ctx.Input.Session(beeC["sessionname"]).(map[string]interface{})
	if !ok && (rawurl.Path != "/user/login/console" && rawurl.Path != "/user/register/console") {
		log.Printf("DEBUG [*] FilterUser ctx.Input.CruSession: %#v", ctx.Input.CruSession)
		ctx.Redirect(302, "/user/login/console")
	}
}
