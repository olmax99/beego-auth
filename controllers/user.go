package controllers

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"time"

	"beego-auth/conf"
	"beego-auth/models"

	"github.com/astaxie/beego/validation"
	"github.com/beego/beego/v2/adapter/orm"
	"github.com/twinj/uuid"

	beego "github.com/beego/beego/v2/server/web"
	vault "github.com/olmax99/vaultgo"
	sendgrid "github.com/sendgrid/sendgrid-go"
	mail "github.com/sendgrid/sendgrid-go/helpers/mail"

	// TODO replace pk in favor of hashicorp/vault
	pk "beego-auth/utilities/pbkdf2"
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

		dbalias, err := beego.AppConfig.String("db::beego_db_alias")
		if err != nil {
			fmt.Printf("ERROR [*] Index.dbalias.. %v", err)
		}
		o := orm.NewOrm()
		o.Using(dbalias)
		user := models.AuthUser{Email: email}
		err = o.Read(&user, "Email")
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
	// ----------------------------GET ---------------------------------------------
	this.activeContent("user/register")

	// -----------------------------POST -------------------------------------------
	if this.Ctx.Input.Method() == "POST" {
		// Step 1: -------------- Validate Form input---------------------------
		flash := beego.NewFlash()

		beeC := conf.BeeConf(
			"httpport",
			"db::beego_db_alias",
			"vault::beego_vault_address",
			"vault::beego_vault_token",
			"sendgrid::beego_sg_own_support",
			"sendgrid::beego_sg_api_key",
		)

		first := this.GetString("first")
		last := this.GetString("last")
		email := this.GetString("email")
		password := this.GetString("password")
		password2 := this.GetString("password2")

		// TODO implement https://gowalker.org/github.com/astaxie/beego/validation#ValidFormer:
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
			return
		}
		if password != password2 {
			flash.Error("Passwords don't match")
			flash.Store(&this.Controller)
			return
		}

		// Step 2: -----------Get Vault Client--------------------------
		client, err := vault.NewClient(beeC["beego_vault_address"], vault.WithCaPath(""))
		if err != nil {
			log.Fatal("PANIC [-] Could not connect with Vault")
		}

		// Step 3: ----------- Check for token--------------------------
		if t := client.Token(); t == "" {
			log.Print("WARNING [*] No token found in environment. Try config..")
			client.SetToken(beeC["beego_vault_token"])
		}

		// Step 4: ----------- Encrypt String and Write to DB----------
		trans := client.Transit()
		err = trans.Create("beego-key", &vault.TransitCreateOptions{})
		if err != nil {
			log.Printf("ERROR [*] Could not create Vault client.. %v", err)
		}
		// Step 5: -------------- Save user info to database--------------------
		o := orm.NewOrm()
		o.Using(beeC["beego_db_alias"])

		user := new(models.AuthUser)
		user.First = first
		user.Last = last
		user.Email = email
		enc, err := trans.Encrypt("beego-key", &vault.TransitEncryptOptions{
			Plaintext: password2,
		})
		if err != nil {
			log.Printf("ERROR [*] Encrypt failed.. %v", err)
		}
		ps := string(enc.Data.Ciphertext)
		user.Password = ps

		// Add user to database with new uuid and
		u := uuid.NewV4()
		user.Reg_key = u.String()
		_, err = o.Insert(user)
		if err != nil {
			// TODO confirm if other errors need to be handled??
			// TODO This Error returns blank page <- capture error notice
			flash.Error(email + " already registered")
			flash.Store(&this.Controller)
			return
		}

		// Step 6: --------------- Send verification email----------------------
		// TODO Verify successfully send (webhook??)
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

// TODO Creating Email templates might go into a separate package or folder within controller
// verification email after user registered
func sendVerification(authusr *models.AuthUser, uid string, conf map[string]string) bool {
	// Step 1: -------------------- Prepare html---------------------------------
	log.Printf("INFO [*] sendVerification link: %v", conf["httpport"])
	link := "http://localhost:" + conf["httpport"] + "/user/verify/" + uid
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	t, err := template.New("verification.tpl").ParseFiles(pwd + "/controllers/html/verification.tpl")
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
	subject := "Sending with SendGrid: html content"
	to := mail.NewEmail("Peter Pan", string(authusr.Email))
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

func (this *MainController) Verify() {
	// ----------------------------- GET--------------------------------------------
	this.activeContent("user/verify")

	beeC := conf.BeeConf(
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
		fmt.Println(err)
	}
}

func (this *MainController) Profile() {
	// ----------------------------- GET--------------------------------------------
	this.activeContent("user/profile")

	//////////////////////////////
	// This page requires login //
	//////////////////////////////
	sess := this.GetSession("auth")
	if sess == nil {
		this.Redirect("/user/login/home", 302)
		return
	}
	m := sess.(map[string]interface{})

	// Step 1:---------- Read current password hash from database-----------------
	flash := beego.NewFlash()

	// TODO redo with Vault logic
	var x pk.PasswordHash

	x.Hash = make([]byte, 32)
	x.Salt = make([]byte, 16)

	o := orm.NewOrm()
	o.Using("authdb")
	user := models.AuthUser{Email: m["username"].(string)}
	err := o.Read(&user, "Email")
	if err == nil {
		// scan in the password hash/salt (see POST 'Step 2' for current
		// password validation
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

	// ----------------------------- POST-------------------------------------------
	// Profile can be used to update user profile data
	// Profile is providing change user password functionality
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
			valid.MinSize(password, 12, "password")
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

		// TODO redo logic for using Vault
		// Step 2:---------- Compare submitted password with database---------
		// Ensure that controller drops out if current password does not match
		// with DB
		if !pk.MatchPassword(current, &x) {
			flash.Error("Bad current password")
			flash.Store(&this.Controller)
			return
		}

		// Step 3: --------- update user info in db---------------------------
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

	// explicit render (can be omitted by setting: 'autorender = true')
	errR := this.Render()
	if err != nil {
		fmt.Println(errR)
	}
}

func (this *MainController) Remove() {
	// ----------------------------- GET------------------------------------------
	this.activeContent("user/remove")

	//////////////////////////////
	// This page requires login //
	//////////////////////////////
	sess := this.GetSession("acme")
	if sess == nil {
		this.Redirect("/user/login/home", 302)
		return
	}
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
		// User is required to provide password in order to proceed
		var x pk.PasswordHash

		x.Hash = make([]byte, 32)
		x.Salt = make([]byte, 16)

		o := orm.NewOrm()
		o.Using("default")
		user := models.AuthUser{Email: m["username"].(string)}
		err := o.Read(&user, "Email")
		if err == nil {
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

		// Step 2: -------- Compare submitted password with database------
		if !pk.MatchPassword(current, &x) {
			flash.Error("Bad current password")
			flash.Store(&this.Controller)
			return
		}

		// Step 3:--------- Delete user record----------------------------
		_, err = o.Delete(user)
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

	// explicit render (can be omitted by setting: 'autorender = true')
	err := this.Render()
	if err != nil {
		fmt.Println(err)
	}
}
