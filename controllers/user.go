package controllers

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"time"

	"beego-auth/conf"
	"beego-auth/models"

	"github.com/beego/beego/v2/adapter/orm"
	"github.com/beego/beego/v2/adapter/utils"
	"github.com/beego/beego/v2/core/validation"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/twinj/uuid"

	beego "github.com/beego/beego/v2/server/web"
	sendgrid "github.com/sendgrid/sendgrid-go"
	mail "github.com/sendgrid/sendgrid-go/helpers/mail"
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

		flash := beego.NewFlash()

		beeC := conf.BeeConf("",
			"sessionname",
			"db::beego_db_alias",
			"vault::beego_vault_transit_key",
			"vault::beego_vault_address",
			"vault::beego_vault_token",
		)

		// Step 1: -----------Validate Form input--------------------------------
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
			errR := this.Render()
			if errR != nil {
				fmt.Println(errR)
			}
			return
		}

		// Step 2: ----------- Read User Password from DB and Decrypt----------
		crypt := NewCrypter(beeC)
		user := this.ReadAuthUser(email, beeC["beego_db_alias"])()

		// Verify() will remove uuid from user, hence if it still exists
		// it indicates that account verification (email) has not been
		// completed
		if user.Reg_key != "" || user.Active != "" {
			flash.Error("Account not active.")
			flash.Store(&this.Controller)
			errR := this.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}
		// TODO move to crypt.go
		// Step 5: ----------- Compare password with db--------
		crypt.vaultv1 = user.Password
		if err := crypt.De(); err != nil {
			log.Printf("ERROR [*] VaultDecryptVal.Set().. %v", err)
		}
		if crypt.value != password {
			flash.Error("Bad password")
			flash.Store(&this.Controller)
			errR := this.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}
		// Step 6: ------------ Create session and go back to previous page------
		m := make(map[string]interface{})
		m["first"] = user.First
		m["username"] = email
		m["timestamp"] = time.Now()
		this.SetSession(beeC["sessionname"], m)
		this.Redirect("/"+back, 302)
	}
	errR := this.Render()
	if errR != nil {
		fmt.Println(errR)
	}
}

func (this *MainController) Logout() {
	flash := beego.NewFlash()

	beeC := conf.BeeConf("",
		"sessionname",
	)

	this.activeContent("user/logout")
	this.DelSession(beeC["sessionname"])

	flash.Notice("Thanks for checking in, Bye.")
	flash.Store(&this.Controller)
	this.Redirect("/notice", 302)
}

func (this *MainController) Register() {
	// ----------------------------GET ---------------------------------------------
	this.activeContent("user/register")

	// -----------------------------POST -------------------------------------------
	if this.Ctx.Input.Method() == "POST" {
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

		first := this.GetString("first")
		last := this.GetString("last")
		email := this.GetString("email")
		password := this.GetString("password")
		password2 := this.GetString("password2")

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
			this.Data["Errors"] = errormap
			errR := this.Render()
			if errR != nil {
				fmt.Printf("ERROR [*] MainController.Register() validation.. %v", errR)
			}
		}
		if password != password2 {
			flash.Error("Passwords don't match")
			flash.Store(&this.Controller)
			return
		}
		// Step 5: -------------- Save user info to database--------------------
		u := uuid.NewV4() // new user verify uuid

		crypt := NewCrypter(beeC)
		crypt.value = password2
		if err := crypt.En(); err != nil {
			log.Printf("ERROR [*] MainController.Register(), VaultCrypter.En().. %v", err)
		}

		o := orm.NewOrm()
		o.Using(beeC["beego_db_alias"])
		user := new(models.AuthUser)
		user.First = first
		user.Last = last
		user.Email = email
		user.Password = crypt.vaultv1
		user.Reg_key = u.String()

		_, err := o.Insert(user)
		if err != nil {
			log.Printf("ERROR [*] Register() Insert.. %v", err)
			// TODO: confirm if other errors need to be handled??
			flash.Error(email + " already registered")
			flash.Store(&this.Controller)
			errR := this.Render()
			if errR != nil {
				fmt.Printf("ERROR [*] MainController.Register() save user.. %v", errR)
			}

		}

		// Step 6: --------------- Send verification email----------------------
		// TODO: Verify successfully send (webhook??)
		if !sendVerification(user, u.String(), beeC) {
			flash.Error("Unable to send verification email")
			flash.Store(&this.Controller)
			return
		}

		// Step 7: --------------- Append confirmation to flash & redirect------
		flash.Notice("Your account has been created. You must verify the account in your email.")
		flash.Store(&this.Controller)
		this.Redirect("/notice", 302)
	}

	// explicit render (can be omitted by setting: 'autorender = true')
	err := this.Render()
	if err != nil {
		fmt.Println(err)
	}
}

func (this *MainController) Verify() {
	// ----------------------------- GET--------------------------------------------
	this.activeContent("user/verify")

	beeC := conf.BeeConf("",
		"db::beego_db_alias",
	)

	u := this.Ctx.Input.Param(":uuid")
	o := orm.NewOrm()
	o.Using(beeC["beego_db_alias"])

	// Get user from data base by filtering on uuid
	user := &models.AuthUser{Reg_key: u}
	err := o.Read(user, "Reg_key")
	if err == nil {
		this.Data["Verified"] = 1
		// Remove registration key after context has 'Verified=1'
		user.Reg_key = ""
		if _, err := o.Update(user); err != nil {
			delete(this.Data, "Verified")
		}
	}

	// explicit render (can be omitted by setting: 'autorender = true')
	errR := this.Render()
	if errR != nil {
		fmt.Println(errR)
	}
}

func (this *MainController) Cancel() {
	// ----------------------------- GET--------------------------------------------
	this.activeContent("user/cancel")

	flash := beego.NewFlash()

	beeC := conf.BeeConf("",
		"db::beego_db_alias",
	)

	u := this.Ctx.Input.Param(":uuid")

	o := orm.NewOrm()
	o.Using(beeC["beego_db_alias"])
	user := &models.AuthUser{Active: u}
	err := o.Read(user, "Active")
	if err == nil && user.Active == u {
		this.Data["Cancelled"] = 1
		user.Clc_date = time.Now().UTC()
		if _, err := o.Update(user); err != nil {
			log.Printf("Error [*] Cancel failed.. %v", err)
			flash.Error("Deactivation failed. Please try again..")
			flash.Store(&this.Controller)
			errR := this.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}
		delete(this.Data, "Cancelled")
		this.DelSession(beeC["sessionname"])
		errR := this.Render()
		if errR != nil {
			fmt.Println(errR)
		}
	}
	flash.Error("Deactivation failed. Please try again..")
	flash.Store(&this.Controller)
	errR := this.Render()
	if errR != nil {
		fmt.Println(errR)
	}
}

func (this *MainController) Genpass1() {
	// ----------------------------- GET--------------------------------------------
	// TODO: Improve by both reading uuid and allow user to post his current password
	// At this point it was not clear on how to combine those two
	flash := beego.NewFlash()

	beeC := conf.BeeConf("",
		"sessionname",
		"db::beego_db_alias",
	)
	u := this.Ctx.Input.Param(":uuid")

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
			flash.Store(&this.Controller)
			this.Redirect("/notice", 302)
		}
		// Step 2: ----------- Create new session---------------------------
		m := make(map[string]interface{})
		m["first"] = user.First
		m["username"] = user.Email
		m["timestamp"] = time.Now()
		this.SetSession(beeC["sessionname"], m)
		this.Redirect("/user/genpass2", 307)
	}
	flash.Notice("You can now login with your temporary password.")
	flash.Store(&this.Controller)
	this.Redirect("/notice", 302)
}

func (this *MainController) Genpass2() {
	this.activeContent("user/genpass2")

	flash := beego.NewFlash()

	beeC := conf.BeeConf("",
		"sessionname",
		"db::beego_db_alias",
		"vault::beego_vault_address",
		"vault::beego_vault_token",
		"vault::beego_vault_transit_key",
	)

	sess := this.GetSession(beeC["sessionname"])
	m := sess.(map[string]interface{})

	// Step 1:---------- Read current password hash from database-----------------
	crypt := NewCrypter(beeC)
	user := this.ReadAuthUser(m["username"].(string), beeC["beego_db_alias"])()
	crypt.vaultv1 = user.Password
	if err := crypt.De(); err != nil {
		log.Printf("ERROR [*] VaultDecrypter.Set().. %v", err)
	}

	if this.Ctx.Input.Method() == "POST" {
		current := this.GetString("current")
		password := this.GetString("password")
		password2 := this.GetString("password2")
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
			this.Data["Errors"] = errormap
			errR := this.Render()
			if errR != nil {
				fmt.Printf("ERROR [*] MainController.Register() validation.. %v", errR)
			}
		}
		if password != password2 {
			flash.Error("Passwords don't match.")
			flash.Store(&this.Controller)
			errR := this.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}
		if crypt.value != current {
			flash.Error("The current password is incorrect.")
			flash.Store(&this.Controller)
			errR := this.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}
		// Step 3: ---------- Update Password in database----------------------
		crypt.value = password
		if err := crypt.En(); err != nil {
			log.Printf("ERROR [*] MainController.Genpass2(), VaultCrypter.En().. %v", err)
		}
		user.Password = crypt.vaultv1
		o := orm.NewOrm()
		o.Using(beeC["beego_db_alias"])
		_, err := o.Update(user)
		if err != nil {
			flash.Error("Password reset failed. Please try again..")
			flash.Store(&this.Controller)
			errR := this.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}
		flash.Notice("Your temporary password has been updated.")
		flash.Store(&this.Controller)
		this.Redirect("/notice", 302)
	}
	errR := this.Render()
	if errR != nil {
		fmt.Println(errR)
	}
}

func (this *MainController) Profile() {
	// ----------------------------- GET--------------------------------------------
	this.activeContent("user/profile")

	flash := beego.NewFlash()

	beeC := conf.BeeConf("",
		"sessionname",
		"db::beego_db_alias",
		"vault::beego_vault_address",
		"vault::beego_vault_token",
		"vault::beego_vault_transit_key",
	)

	sess := this.GetSession(beeC["sessionname"])
	m := sess.(map[string]interface{})

	// Step 1:---------- Read current password hash from database-----------------
	crypt := NewCrypter(beeC)
	user := this.ReadAuthUser(m["username"].(string), beeC["beego_db_alias"])()
	if user.Reg_key != "" {
		flash.Error("Account not verified")
		flash.Store(&this.Controller)
		return
	}
	crypt.vaultv1 = user.Password
	if err := crypt.De(); err != nil {
		log.Printf("ERROR [*] VaultDecrypter.Set().. %v", err)
	}
	this.Data["First"] = user.First
	this.Data["Last"] = user.Last
	this.Data["Email"] = user.Email

	// this deferred function ensures that the correct fields from the database are displayed
	defer func(this *MainController, user *models.AuthUser) {
		this.Data["First"] = user.First
		this.Data["Last"] = user.Last
		this.Data["Email"] = user.Email
	}(this, user)

	// ----------------------------- POST-------------------------------------------
	// Profile can be used to update user profile data
	// Profile is providing change user password functionality
	if this.Ctx.Input.Method() == "POST" {
		first := this.GetString("first")
		last := this.GetString("last")
		email := this.GetString("email")
		current := this.GetString("current")
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
			errR := this.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}
		if email != this.Data["Email"] {
			flash.Error("Your Email cannot be changed. You must create a new account.")
			flash.Store(&this.Controller)
			errR := this.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}

		// Step 2:---------- Compare submitted password with database---------
		// Ensure that controller drops out if current password does not match
		// with DB
		if current != crypt.value {
			flash.Error("Bad current password")
			flash.Store(&this.Controller)
			errR := this.Render()
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
			flash.Store(&this.Controller)
			m["username"] = email
		} else {
			flash.Error("Internal error")
			flash.Store(&this.Controller)
			return
		}
	}

	// explicit render (can be omitted by setting: 'autorender = true')
	errR := this.Render()
	if errR != nil {
		fmt.Println(errR)
	}
}

func (this *MainController) Remove() {
	// ----------------------------- GET------------------------------------------
	this.activeContent("user/remove")

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

	sess := this.GetSession(beeC["sessionname"])
	m := sess.(map[string]interface{})

	// -----------------------------POST -----------------------------------------
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

		// Step 1---------- Read password hash from database-------------
		crypt := NewCrypter(beeC)
		user := this.ReadAuthUser(m["username"].(string), beeC["beego_db_alias"])()
		// Verify() will remove uuid from user, hence if it still exists
		// it indicates that account verification (email) has not been
		// completed
		if user.Reg_key != "" {
			flash.Error("Account not verified")
			flash.Store(&this.Controller)
			return
		}
		crypt.vaultv1 = user.Password
		if err := crypt.De(); err != nil {
			log.Printf("ERROR [*] VaultDecryptVal.Set().. %v", err)
		}
		// User is required to provide password in order to proceed
		// Step 2: -------- Compare submitted password with database------
		if current != crypt.value {
			flash.Error("Bad current password")
			flash.Store(&this.Controller)
			errR := this.Render()
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
			flash.Store(&this.Controller)
			errR := this.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}

		// Step 3:--------- Delete user record----------------------------
		// Send email
		if !sendCancellation(user, a.String(), beeC) {
			flash.Error("Unable to send cancellation email")
			flash.Store(&this.Controller)
			errR := this.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}

		// Step 7: --------------- Append confirmation to flash & redirect------
		flash.Notice("We are sad to see you go. You will receive an email soon with instructions on how to cancel your account.")
		flash.Store(&this.Controller)
		this.Redirect("/notice", 302)
	}

	// explicit render (can be omitted by setting: 'autorender = true')
	err := this.Render()
	if err != nil {
		fmt.Println(err)
	}
}

func (this *MainController) Reset() {
	this.activeContent("user/reset")

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
	if this.Ctx.Input.Method() == "POST" {
		current := this.GetString("current")
		valid := validation.Validation{}
		valid.Email(current, "current")
		valid.Required(current, "current")
		if valid.HasErrors() {
			errormap := []string{}
			for _, err := range valid.Errors {
				errormap = append(errormap, "Validation failed on "+err.Key+": "+err.Message+"\n")
			}
			this.Data["Errors"] = errormap
			errR := this.Render()
			if errR != nil {
				fmt.Printf("ERROR [*] MainController.Reset() validation.. %v", errR)
			}
		}

		flash := beego.NewFlash()

		// Step 1---------- Read password hash from database-------------
		user := this.ReadAuthUser(current, beeC["beego_db_alias"])()
		// Drop out if last password reset was within 24h
		// The first time an empty time field is accepted
		if !user.Rst_date.Equal(time.Time{}) {
			tn := time.Now().UTC()
			tr := user.Rst_date
			if 24*time.Hour > tn.Sub(tr) {
				flash.Error("There must be at least 24h after your last reset.")
				flash.Store(&this.Controller)
				errR := this.Render()
				if errR != nil {
					fmt.Println(errR)
				}
			}
		}
		// Step 2: ------ Generate New Random Password----------------
		r := uuid.NewV4()
		newPass := utils.RandomCreateBytes(16)
		crypt := NewCrypter(beeC)
		crypt.value = string(newPass)
		if err := crypt.En(); err != nil {
			log.Printf("ERROR [*] MainController.Register(), VaultCrypter.En().. %v", err)
		}
		user.Reset = r.String()
		user.Password = crypt.vaultv1
		o := orm.NewOrm()
		o.Using(beeC["beego_db_alias"])
		_, err := o.Update(user)
		if err != nil {
			flash.Error("Deactivation failed. Please try again..")
			flash.Store(&this.Controller)
			errR := this.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}

		// Step 3: ------ Send Reset Email-----------------------------
		if !sendReset(user, r.String(), newPass, beeC) {
			flash.Error("Unable to send password reset email")
			flash.Store(&this.Controller)
			errR := this.Render()
			if errR != nil {
				fmt.Println(errR)
			}
		}

		// Step 4: ------ Append confirmation to flash & redirect------
		flash.Notice("Your password has been reset. Please follow the instructions in your confirmation email.")
		flash.Store(&this.Controller)
		this.Redirect("/notice", 302)
	}

	// explicit render (can be omitted by setting: 'autorender = true')
	errR := this.Render()
	if errR != nil {
		fmt.Println(errR)
	}
}

// TODO move to model - needs separation from controller
// ReadAuthUser ensures that subsequent actions do have an existing user struct pointer
// and if not differentiates the Read errors
func (c *MainController) ReadAuthUser(se, al string) func() *models.AuthUser {
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

// TODO move to sendgrid package
// verification email after user registered
func sendVerification(authusr *models.AuthUser, uid string, conf map[string]string) bool {
	// Step 1: -------------------- Prepare html---------------------------------
	link := "http://localhost:" + conf["httpport"] + "/user/verify/" + uid
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	t, err := template.New("verification.tpl").ParseFiles(pwd + "/controllers/verification.tpl")
	if err != nil {
		log.Printf("ERROR [*] Parse html failed.. %v", err)
	}
	data := struct {
		User string
		Link string
	}{
		User: string(authusr.First),
		Link: link,
	}

	var tplOut bytes.Buffer
	if err := t.Execute(&tplOut, data); err != nil {
		log.Printf("ERROR [*] sendVerification: tpl.exec failed.. %v", err)
		return false
	}
	content := tplOut.String()

	// Step 2: -------------------- Send Email-----------------------------------
	from := mail.NewEmail("Your Support Team", conf["beego_sg_own_support"])
	subject := "Sending with SendGrid: account verification"
	to := mail.NewEmail("Peter Pan", authusr.Email)
	htmlContent := content
	message := mail.NewSingleEmail(from, subject, to, "", htmlContent)

	// -> /v3/mail/send
	// TODO: This does not catch response errors, i.e. Request successful, but
	// error within sendgrid ('the from address does not match a verified Sender
	// Identity')
	sg_client := sendgrid.NewSendClient(conf["beego_sg_api_key"])
	response, err := sg_client.Send(message)
	if err != nil {
		log.Printf("ERROR [*] Sending email failed.. %v", err)
		return false
	} else {
		log.Printf("INFO [*] Sendgrid response, status: %v", response.StatusCode)
		log.Printf("INFO [*] Sendgrid response, body: %v", response.Body)
		log.Printf("INFO [*] Sendgrid response, header: %#v", response.Headers)
	}
	return true
}

// verification email after user cancelled
func sendCancellation(authusr *models.AuthUser, uid string, conf map[string]string) bool {
	// Step 1: -------------------- Prepare html---------------------------------
	link := "http://localhost:" + conf["httpport"] + "/user/cancel/" + uid
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	t, err := template.New("cancellation.tpl").ParseFiles(pwd + "/controllers/cancellation.tpl")
	if err != nil {
		log.Printf("ERROR [*] Parse html failed.. %v", err)
	}
	data := struct {
		User string
		Link string
	}{
		User: string(authusr.First),
		Link: link,
	}

	var tplOut bytes.Buffer
	if err := t.Execute(&tplOut, data); err != nil {
		log.Printf("ERROR [*] sendCancellation: tpl.exec failed.. %v", err)
		return false
	}
	content := tplOut.String()

	// Step 2: -------------------- Send Email-----------------------------------
	from := mail.NewEmail("Your Support Team", conf["beego_sg_own_support"])
	subject := "Sending with SendGrid: account cancellation"
	to := mail.NewEmail("Peter Pan", authusr.Email)
	htmlContent := content
	message := mail.NewSingleEmail(from, subject, to, "", htmlContent)

	// -> /v3/mail/send
	sg_client := sendgrid.NewSendClient(conf["beego_sg_api_key"])
	response, err := sg_client.Send(message)
	if err != nil {
		log.Printf("ERROR [*] Sending email failed.. %v", err)
		return false
	} else {
		log.Printf("INFO [*] Sendgrid response, status: %v", response.StatusCode)
		log.Printf("INFO [*] Sendgrid response, body: %v", response.Body)
		log.Printf("INFO [*] Sendgrid response, header: %#v", response.Headers)
	}
	return true
}

// verification email after user initiated to reset password
func sendReset(authusr *models.AuthUser, uid string, pass []byte, conf map[string]string) bool {
	// Step 1: -------------------- Prepare html---------------------------------
	link := "http://localhost:" + conf["httpport"] + "/user/genpass1/" + uid
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	t, err := template.New("reset.tpl").ParseFiles(pwd + "/controllers/reset.tpl")
	if err != nil {
		log.Printf("ERROR [*] Parse html failed.. %v", err)
	}

	data := struct {
		User string
		Link string
		RanB string
	}{
		User: string(authusr.First),
		Link: link,
		RanB: string(pass),
	}

	var tplOut bytes.Buffer
	if err := t.Execute(&tplOut, data); err != nil {
		log.Printf("ERROR [*] sendReset: tpl.exec failed.. %v", err)
		return false
	}
	content := tplOut.String()

	// Step 2: -------------------- Send Email-----------------------------------
	from := mail.NewEmail("Your Support Team", conf["beego_sg_own_support"])
	subject := "Sending with SendGrid: password reset"
	to := mail.NewEmail("Peter Pan", authusr.Email)
	htmlContent := content
	message := mail.NewSingleEmail(from, subject, to, "", htmlContent)

	// -> /v3/mail/send
	sg_client := sendgrid.NewSendClient(conf["beego_sg_api_key"])
	response, err := sg_client.Send(message)
	if err != nil {
		log.Printf("ERROR [*] Sending email failed.. %v", err)
		return false
	} else {
		log.Printf("INFO [*] Sendgrid response, status: %v", response.StatusCode)
		log.Printf("INFO [*] Sendgrid response, body: %v", response.Body)
		log.Printf("INFO [*] Sendgrid response, header: %#v", response.Headers)
	}
	return true
}

// TODO mode to model, create non-Beego related user tasks (combine with ReadAuthUser)
func doStuffWithAuthUser(u *models.AuthUser) error {
	log.Printf("INFO [*] do something with user: %#v", u)
	return nil
}

// customize filters for fine grain authorization
var FilterUser = func(ctx *context.Context) {
	beeC := conf.BeeConf("",
		"sessionName",
	)
	// TODO: evaluate if this is method is restrictive enough..
	// Do not authorize when:
	// 1. a session does not exists
	// 2. the request does not come from Request.RequestURI "/login"
	// 3. the request does not come from Request.RequestURI "/register"
	_, ok := ctx.Input.Session(beeC["sessionName"]).(int)
	if !ok && ctx.Input.URI() != "/user/login" && ctx.Input.URI() != "/user/register" {
		fmt.Printf("DEBUG [*] FilterUser ctx.Input.CruSession: %#v", ctx.Input.CruSession)
		ctx.Redirect(302, "/user/login/home")
	}
}
