package controllers

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"beego-auth/conf"
	"beego-auth/models"

	sg "beego-auth/pkg/sendgridv1"
	crypt "beego-auth/pkg/vaultv1"

	"github.com/beego/beego/v2/adapter/utils"
	"github.com/beego/beego/v2/core/validation"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/twinj/uuid"

	beego "github.com/beego/beego/v2/server/web"
)

// TODO: separate all beego stuff: no need for testing
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
				log.Printf("ERROR [*] Login().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
		}
		u := models.NewUser(beeC["beego_db_alias"], models.WithEmail(email))
		if err := u.UserConfirm("Email"); err != nil {
			log.Printf("DEBUG [*] u.UserConfirm: %v", err)
			flash.Error("No such user.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Login()User.confirm().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}
		// Verify() will remove uuid from user.Reg_key, hence if it still exists
		// it indicates that account verification (email) has not been completed
		if u.UserActive() {
			flash.Error("Account not active.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Login()User.confirm().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}
		// Step 2: ------ Compare password with db------------------------
		// TODO: Try handle dependency injection with google/wire (user -> crypt)
		crypt := crypt.NewCrypter(beeC)
		if err := crypt.SetVaultv1(u.GetPasswd()); err != nil {
			log.Printf("ERROR [*] Login().VaultDecryptVal.Set().. %v", err)
			flash.Error("Setting password failed. Please try again.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Login(). ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}
		if !crypt.Match(password) {
			flash.Error("Bad password.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Login(). ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}
		// Step 3: ------ Create session and go back to previous page------
		m := make(map[string]interface{})
		m["first"] = u.GetFirst()
		m["username"] = email
		m["timestamp"] = time.Now()
		ctl.SetSession(beeC["sessionname"], m)
		ctl.Redirect("/"+back, 302)
		// Usually put return after redirect.
		return
	}
	if err := ctl.Render(); err != nil {
		log.Println(err)
	}
}

func (ctl *DefaultController) Logout() {
	flash := beego.NewFlash()

	beeC := conf.BeeConf("",
		"sessionname",
	)

	ctl.activeContent("user/logout")
	ctl.DelSession(beeC["sessionname"])

	flash.Notice("Thanks for checking in. Bye.")
	flash.Store(&ctl.Controller)
	ctl.Redirect("/notice", 302)
	return
}

func (ctl *DefaultController) Register() {
	// ----------------------------GET ---------------------------------------------
	ctl.activeContent("user/register")

	// -----------------------------POST -------------------------------------------
	if ctl.Ctx.Input.Method() == "POST" {
		flash := beego.NewFlash()

		beeC := conf.BeeConf("",
			"httpport",
			"baseurl",
			"sessionname",
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
				log.Printf("ERROR [*] Register().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Printf("ERROR [*] DefaultController.Register() validation.. %v", err)
			}
		}
		if password != password2 {
			flash.Error("Passwords don't match")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Register().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Printf("ERROR [*] DefaultController.Register() validation.. %v", err)
			}
			return
		}
		// Step 2: -------------- Save user info to database--------------------
		crypt := crypt.NewCrypter(beeC)
		if err := crypt.SetValue(password2); err != nil {
			log.Printf("ERROR [*] DefaultController.Register(), VaultCrypter.en().. %v", err)
			flash.Error("Encrypting password failed. Please try again.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Register().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Printf("ERROR [*] DefaultController.Register() save user.. %v", err)
			}
			return
		}
		uid4 := uuid.NewV4() // new user verify uuid
		user := models.NewUser(beeC["beego_db_alias"],
			models.PrepareWrite(first, last, email, crypt.GetVaultv1(), uid4.String()))
		err := user.UserInsertTx()
		if err != nil {
			log.Printf("ERROR [*] Register() Insert.. %v", err)
			// TODO: confirm if other errors need to be handled??
			flash.Error("User with email" + email + " already exists.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Register().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Printf("ERROR [*] DefaultController.Register() save user.. %v", err)
			}
			return

		}

		// Step 3: --------------- Send verification email----------------------
		register_link := beeC["baseurl"] + ":" + beeC["httpport"] + "/user/verify/" + uid4.String()
		sg_mail := sg.NewSgSendMail(
			beeC["beego_sg_api_key"], beeC["beego_sg_own_support"], "",
			sg.MailTo(user.GetFirst(), user.GetEmail()),
			sg.PrepareHtml(user.GetFirst(), register_link, "verification.tpl"),
		)
		if res := sg_mail.Send("Verify your new account."); res.StatusCode != 202 {
			log.Printf("DEBUG [*] SgSendMailVerification.. %#v", res)
			flash.Error("Unable to send verification email.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Register().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Printf("ERROR [*] DefaultController.Register() send verification email.. %v", err)
			}
			return
		}

		// Step 4: --------------- Append confirmation to flash & redirect------
		flash.Notice("Your account has been created. An email is on the way to you. Please verify your address by following the link.")
		flash.Store(&ctl.Controller)
		ctl.Redirect("/notice", 302)
		return
	}

	// explicit render (can be omitted by setting: 'autorender = true')
	if err := ctl.Render(); err != nil {
		log.Println(err)
	}
}

func (ctl *DefaultController) Verify() {
	// ----------------------------- GET--------------------------------------------
	ctl.activeContent("user/verify")

	flash := beego.NewFlash()

	beeC := conf.BeeConf("",
		"sessionname",
		"db::beego_db_alias",
	)

	uid4 := ctl.Ctx.Input.Param(":uuid")

	// Get user from data base by filtering on uuid
	user := models.NewUser(beeC["beego_db_alias"], models.WithRegKey(uid4))
	if err := user.UserConfirm("Reg_key"); err != nil {
		log.Printf("DEBUG [*] u.UserConfirm: %v", err)
		flash.Error("No user identified with this registration key.")
		flash.Store(&ctl.Controller)
		if err := ctl.DelSession(beeC["sessionname"]); err != nil {
			log.Printf("ERROR [*] Verify().ctl.DelSession().. %s", err)
		}
		if err := ctl.Render(); err != nil {
			log.Println(err)
		}
		return
	}
	ctl.Data["Verified"] = 1
	// Remove registration key after context has 'Verified=1'
	if err := user.UserRemoveRegKey(); err != nil {
		delete(ctl.Data, "Verified")
		if err := ctl.DelSession(beeC["sessionname"]); err != nil {
			log.Printf("ERROR [*] Verify().ctl.DelSession().. %s", err)
		}
	}
	// explicit render (can be omitted by setting: 'autorender = true')
	if err := ctl.Render(); err != nil {
		log.Println(err)
	}
}

func (ctl *DefaultController) Cancel() {
	// ----------------------------- GET--------------------------------------------
	ctl.activeContent("user/cancel")

	flash := beego.NewFlash()

	beeC := conf.BeeConf("",
		"sessionname",
		"db::beego_db_alias",
	)

	uid4 := ctl.Ctx.Input.Param(":uuid")
	user := models.NewUser(beeC["beego_db_alias"], models.WithActive(uid4))
	if err := user.UserConfirm("Active"); err != nil {
		flash.Error("No active user identified.")
		flash.Store(&ctl.Controller)
		if err := ctl.DelSession(beeC["sessionname"]); err != nil {
			log.Printf("ERROR [*] Cancel().ctl.DelSession().. %s", err)
		}
		if err := ctl.Render(); err != nil {
			log.Println(err)
		}
		return
	}

	if user.GetActive() == uid4 {
		ctl.Data["Cancelled"] = 1
		if err := user.UserUpdateClcDte(time.Now().UTC()); err != nil {
			flash.Error("Deactivation failed. Please try again..")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Cancel().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}
		delete(ctl.Data, "Cancelled")
		ctl.DestroySession()
		if err := ctl.Render(); err != nil {
			log.Println(err)
		}
	}
	flash.Error("Deactivating your account failed. Please try again..")
	flash.Store(&ctl.Controller)
	if err := ctl.Render(); err != nil {
		log.Println(err)
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
	uid4 := ctl.Ctx.Input.Param(":uuid")
	user := models.NewUser(beeC["beego_db_alias"], models.WithReset(uid4))
	if err := user.UserConfirm("Reset"); err != nil {
		log.Printf("DEBUG [*] u.UserConfirm: %v", err)
		flash.Error("No active user identified.")
		flash.Store(&ctl.Controller)
		if err := ctl.Render(); err != nil {
			log.Println(err)
		}
		return
	}
	if user.GetReset() == uid4 {
		// set the 24h limit for next passwd reset
		if err := user.UserUpdateRstDte(time.Now().UTC()); err != nil {
			flash.Notice("Genpass1.. Updating reset date failed.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Cancel().ctl.DelSession().. %s", err)
			}
			ctl.Redirect("/notice", 302)
		}
		// Step 2: ----------- Create new session---------------------------
		m := make(map[string]interface{})
		m["first"] = user.GetFirst()
		m["username"] = user.GetEmail()
		m["timestamp"] = time.Now()
		ctl.SetSession(beeC["sessionname"], m)
		ctl.Redirect("/user/genpass2", 307)
		return
	}
	flash.Notice("You can now login with your temporary password.")
	flash.Store(&ctl.Controller)
	ctl.Redirect("/notice", 302)
	return
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
	// user := ctl.ReadAuthUser(m["username"].(string), beeC["beego_db_alias"])()
	user := models.NewUser(beeC["beego_db_alias"], models.WithEmail(m["username"].(string)))
	if err := user.UserConfirm("Email"); err != nil {
		flash.Error("No such user.")
		flash.Store(&ctl.Controller)
		if err := ctl.DelSession(beeC["sessionname"]); err != nil {
			log.Printf("ERROR [*] Genpass2().ctl.DelSession().. %s", err)
		}
		if err := ctl.Render(); err != nil {
			log.Println(err)
		}
		return
	}
	crypt := crypt.NewCrypter(beeC)
	if err := crypt.SetVaultv1(user.GetPasswd()); err != nil {
		flash.Error("Setting password failed. Please try again.")
		flash.Store(&ctl.Controller)
		if err := ctl.DelSession(beeC["sessionname"]); err != nil {
			log.Printf("ERROR [*] Genpass2().ctl.DelSession().. %s", err)
		}
		if err := ctl.Render(); err != nil {
			log.Println(err)
		}
		return
	}

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
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Genpass2().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
		}
		if password != password2 {
			flash.Error("Passwords don't match.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Genpass2().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
		}
		if crypt.GetValue() != current {
			flash.Error("The current password is incorrect.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Genpass2().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}
		// Step 3: ---------- Update Password in database----------------------
		if err := crypt.SetVaultv1(password); err != nil {
			log.Printf("ERROR [*] Genpass2 VaultDecrypter.set().. %v", err)
			flash.Error("Updating Password failed. Please try again..")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Genpass2().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}
		if err := user.UserUpdatePasswd(crypt.GetVaultv1(), ""); err != nil {
			flash.Error("Password reset failed. Please try again..")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Genpass2().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}
		flash.Notice("Your temporary password has been updated.")
		flash.Store(&ctl.Controller)
		ctl.Redirect("/notice", 302)
		return
	}
	errR := ctl.Render()
	if errR != nil {
		log.Println(errR)
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
	user := models.NewUser(beeC["beego_db_alias"], models.WithEmail(m["username"].(string)))
	if err := user.UserConfirm("Email"); err != nil {
		log.Printf("DEBUG [*] u.UserConfirm: %v", err)
		flash.Error("No such user.")
		flash.Store(&ctl.Controller)
		if err := ctl.DelSession(beeC["sessionname"]); err != nil {
			log.Printf("ERROR [*] Profile().ctl.DelSession().. %s", err)
		}
		if err := ctl.Render(); err != nil {
			log.Println(err)
		}
	}
	if user.GetRegkey() != "" {
		flash.Error("Account not verified.")
		flash.Store(&ctl.Controller)
		if err := ctl.DelSession(beeC["sessionname"]); err != nil {
			log.Printf("ERROR [*] Profile().ctl.DelSession().. %s", err)
		}
		if err := ctl.Render(); err != nil {
			log.Println(err)
		}
	}
	if err := crypt.SetVaultv1(user.GetPasswd()); err != nil {
		log.Printf("ERROR [*] Profile() VaultDecrypter.set().. %v", err)
		flash.Error("Setting password failed. Please try again.")
		flash.Store(&ctl.Controller)
		if err := ctl.DelSession(beeC["sessionname"]); err != nil {
			log.Printf("ERROR [*] Profile().ctl.DelSession().. %s", err)
		}
		if err := ctl.Render(); err != nil {
			log.Println(err)
		}
		return
	}
	ctl.Data["First"] = user.GetFirst()
	ctl.Data["Last"] = user.GetLast()
	ctl.Data["Email"] = user.GetEmail()

	// deferred func ensures that the correct fields from the database are displayed
	defer func(ctl *DefaultController, first, last, email string) {
		ctl.Data["First"] = first
		ctl.Data["Last"] = last
		ctl.Data["Email"] = email
	}(ctl, user.GetFirst(), user.GetLast(), user.GetEmail())

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
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Profile().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
		}
		if email != ctl.Data["Email"] {
			flash.Error("Your Email cannot be changed. You need to create a new account.")
			flash.Store(&ctl.Controller)
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
		}

		// Step 2:---------- Compare submitted password with database---------
		// Ensure that controller drops out if current password does not match
		// with DB
		if !crypt.Match(current) {
			flash.Error("Bad current password.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Profile().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}

		// Step 3: --------- update user info in db---------------------------
		if err := user.UserUpdateInfo(first, last); err != nil {
			flash.Error("Internal error.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Profile().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}
		flash.Notice("Profile updated")
		flash.Store(&ctl.Controller)
		m["username"] = email
	}

	// explicit render (can be omitted by setting: 'autorender = true')
	if err := ctl.Render(); err != nil {
		log.Println(err)
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
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Remove().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
		}

		flash := beego.NewFlash()

		// Step 1---------- Read password hash from database-------------
		crypt := crypt.NewCrypter(beeC)
		user := models.NewUser(beeC["beego_db_alias"], models.WithEmail(m["username"].(string)))
		if err := user.UserConfirm("Email"); err != nil {
			log.Printf("DEBUG [*] u.UserConfirm: %v", err)
			flash.Error("No such user.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Remove().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}
		// Verify() will remove uuid from user, hence if it still exists
		// it indicates that account verification (email) has not been
		// completed
		if user.GetRegkey() != "" {
			flash.Error("Account not verified.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Remove().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}
		if err := crypt.SetVaultv1(user.GetPasswd()); err != nil {
			log.Printf("ERROR [*] Remove() VaultDecryptVal.Set().. %v", err)
			flash.Error("Updating Password failed. Please try again..")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Remove().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}

		// User is required to provide password in order to proceed
		// Step 2: -------- Compare submitted password with database------
		if !crypt.Match(current) {
			flash.Error("Bad current password")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Remove().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}

		uid4 := uuid.NewV4()
		if err := user.UserUpdateActive(uid4.String()); err != nil {
			flash.Error("Deactivation failed. Please try again..")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Remove().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}

		// Step 3:--------- Delete user record----------------------------
		cancel_link := beeC["baseurl"] + ":" + beeC["httpport"] + "/user/cancel/" + uid4.String()
		sg_mail := sg.NewSgSendMail(
			beeC["beego_sg_api_key"], beeC["beego_sg_own_support"], "",
			sg.MailTo(user.GetFirst(), user.GetEmail()),
			sg.PrepareHtml(user.GetFirst(), cancel_link, "cancellation.tpl"),
		)
		res := sg_mail.Send("Cancel your account with us.")
		if res.StatusCode != 202 {
			log.Printf("DEBUG [*] SgSendMailVerification.. %#v", res)
			flash.Error("Unable to send verification email.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Remove().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}

		// Step 4: --------------- Append confirmation to flash & redirect------
		flash.Notice("We are sad to see you go. You will receive an email soon with instructions on how to cancel your account.")
		flash.Store(&ctl.Controller)
		ctl.Redirect("/notice", 302)
		return
	}

	// explicit render (can be omitted by setting: 'autorender = true')
	err := ctl.Render()
	if err != nil {
		log.Println(err)
	}
}

func (ctl *DefaultController) Reset() {
	ctl.activeContent("user/reset")

	beeC := conf.BeeConf("",
		"httpport",
		"baseurl",
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
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Reset().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Printf("ERROR [*] DefaultController.Reset() validation.. %v", err)
			}
		}

		flash := beego.NewFlash()

		// Step 1---------- Read password hash from database-------------
		user := models.NewUser(beeC["beego_db_alias"], models.WithEmail(current))
		if err := user.UserConfirm("Email"); err != nil {
			log.Printf("DEBUG [*] u.UserConfirm: %v", err)
			flash.Error("No such user.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Reset().ctl.DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}

		// Drop out if last password reset was within 24h
		// The first time an empty time field is accepted
		if !user.GetResetDte().Equal(time.Time{}) {
			tn := time.Now().UTC()
			tr := user.GetResetDte()
			if 24*time.Hour > tn.Sub(tr) {
				flash.Error("There must be at least 24h after your last reset. Please log in and try again.")
				flash.Store(&ctl.Controller)
				if err := ctl.DelSession(beeC["sessionname"]); err != nil {
					log.Printf("ERROR [*] Reset().GetResetDte().. %s", err)
				}
				if err := ctl.Render(); err != nil {
					log.Println(err)
				}
			}
		} else {
			// Update Rst_date
			if err := user.UserUpdateRstDte(time.Now()); err != nil {
				log.Printf("ERROR [*] u.UserUpdateRstDte: %v", err)
				flash.Error("Failed to update Reset Date. Please log in and try again.")
				flash.Store(&ctl.Controller)
				if err := ctl.DelSession(beeC["sessionname"]); err != nil {
					log.Printf("ERROR [*] Reset().DelSession().. %s", err)
				}
				if err := ctl.Render(); err != nil {
					log.Println(err)
				}
				return
			}

		}
		// Step 2: ------ Generate New Random Password----------------
		uid4 := uuid.NewV4()
		newPass := utils.RandomCreateBytes(16)
		crypt := crypt.NewCrypter(beeC)
		if err := crypt.FromBytes(newPass); err != nil {
			log.Printf("ERROR [*] DefaultController.Register(), VaultCrypter.En().. %v", err)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Reset().DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}

		if err := user.UserUpdatePasswd(crypt.GetVaultv1(), uid4.String()); err != nil {
			flash.Error("Deactivation failed. Please try again..")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Reset().DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}

		// Step 3: ------ Send Reset Email-----------------------------
		reset_link := beeC["baseurl"] + ":" + beeC["httpport"] + "/user/genpass1/" + uid4.String()
		sg_mail := sg.NewSgSendMail(
			beeC["beego_sg_api_key"], beeC["beego_sg_own_support"], "",
			sg.MailTo(user.GetFirst(), user.GetEmail()),
			sg.PrepareHtmlWithReset(user.GetFirst(), reset_link, string(newPass), "reset.tpl"),
		)
		if res := sg_mail.Send("Reset Password of your account."); res.StatusCode != 202 {
			log.Printf("DEBUG [*] SgSendMailVerification.. %#v", res)
			flash.Error("Unable to send password reset email.")
			flash.Store(&ctl.Controller)
			if err := ctl.DelSession(beeC["sessionname"]); err != nil {
				log.Printf("ERROR [*] Reset().DelSession().. %s", err)
			}
			if err := ctl.Render(); err != nil {
				log.Println(err)
			}
			return
		}

		// Step 4: ------ Append confirmation to flash & redirect------
		flash.Notice("Your password has been reset. Please follow the instructions in your confirmation email.")
		flash.Store(&ctl.Controller)
		ctl.Redirect("/notice", 302)
		return
	}

	// explicit render (can be omitted by setting: 'autorender = true')
	if err := ctl.Render(); err != nil {
		log.Println(err)
	}
}

// customize filters for fine grain authorization
var FilterUser = func(ctx *context.Context) {
	beeC := conf.BeeConf("",
		"sessionname",
	)
	// Do NOT authorize when any of the following is true:
	// 1. the request does not come from Request.RequestURI "/login"
	// 2. the request does not come from Request.RequestURI "/register"
	// 3. the session store is nil (default behavior) or contains an empty map
	rawurl, err := url.Parse(ctx.Input.Referer())
	if err != nil {
		log.Printf("DEBUG [*] FilterUser ctx.Input.Referer: %s", err)
	}
	// Session gets the Store content. Store contains all data for one session process
	// with specific id. Type assertion: Session.Store is of type map and not empty
	_, ok := ctx.Input.Session(beeC["sessionname"]).(map[string]interface{})
	if (rawurl.Path != "/user/login/console" && rawurl.Path != "/user/register/console") || !ok {
		log.Printf("DEBUG [*] FilterUser ctx.Input.CruSession: %#v", ctx.Input.CruSession)
		ctx.Redirect(302, "/user/login/console")
	}
}
