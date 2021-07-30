package controllers

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"sync"
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

// Container for retrieving decrypted strings via Vault Transit
type VaultDecryptVal struct {
	vaultv1 string
	value   string
	lock    sync.RWMutex
}

// returns the container with the decrypted user password
func (vdv *VaultDecryptVal) Set(beeCfg map[string]string) error {
	vdv.lock.Lock()
	defer vdv.lock.Unlock()
	client, err := vault.NewClient(beeCfg["beego_vault_address"], vault.WithCaPath(""))
	if err != nil {
		log.Println("PANIC [-] Could not connect with Vault")
		return err
	}

	// Step 4: ----------- Check for token--------------------------
	if t := client.Token(); t == "" {
		log.Println("WARNING [*] No token found in environment. Try config..")
		client.SetToken(beeCfg["beego_vault_token"])
	}

	trans := client.Transit()
	err = trans.Create(beeCfg["beego_vault_transit_key"], &vault.TransitCreateOptions{})
	if err != nil {
		log.Printf("ERROR [*] Could not create Vault client.. %v", err)
		return err
	}
	v, err := trans.Decrypt(beeCfg["beego_vault_transit_key"], &vault.TransitDecryptOptions{
		Ciphertext: vdv.vaultv1,
	})
	vdv.value = v.Data.Plaintext
	if err != nil {
		log.Printf("ERROR [*] Decrypt failed.. %v", err)
		return err
	}
	return nil
}

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

		beeC := conf.BeeConf(
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
			return
		}

		// Step 2: ----------- Read String from DB and Decrypt------------------
		o := orm.NewOrm()
		o.Using(beeC["beego_db_alias"])
		user := &models.AuthUser{Email: email}
		switch err := o.Read(user, "Email"); {
		case err == orm.ErrNoRows:
			log.Printf("ERROR [*] No result found.. %v", err)
			return
		case err == orm.ErrMissPK:
			log.Printf("ERROR [*] No primary key found.. %v", err)
			return
		case err != nil:
			log.Printf("ERROR [*] Something else went wrong.. %v", err)
			return
		case err == nil:
			// Verify() will remove uuid from user, hence if it still exists
			// it indicates that account verification (email) has not been
			// completed
			if user.Reg_key != "" {
				flash.Error("Account not verified")
				flash.Store(&this.Controller)
				return
			}

			// Step 3: -----------Get Vault Client--------------------------
			client, err := vault.NewClient(beeC["beego_vault_address"], vault.WithCaPath(""))
			if err != nil {
				log.Println("PANIC [-] Could not connect with Vault")
			}

			// Step 4: ----------- Check for token--------------------------
			if t := client.Token(); t == "" {
				log.Println("WARNING [*] No token found in environment. Try config..")
				client.SetToken(beeC["beego_vault_token"])
			}

			trans := client.Transit()
			err = trans.Create(beeC["beego_vault_transit_key"], &vault.TransitCreateOptions{})
			if err != nil {
				log.Printf("ERROR [*] Could not create Vault client.. %v", err)
			}
			dec, err := trans.Decrypt(beeC["beego_vault_transit_key"], &vault.TransitDecryptOptions{
				Ciphertext: user.Password,
			})
			if err != nil {
				log.Printf("ERROR [*] Decrypt failed.. %v", err)
			}
			// Step 5: ----------- Compare password with db--------
			if dec.Data.Plaintext != password {
				log.Println("WARNING [*] Login passwords do not match.")
				flash.Error("Bad password")
				flash.Store(&this.Controller)
				return
			}
		default:
			flash.Error("No such user/email")
			flash.Store(&this.Controller)
			return
		}

		// Step 6: ------------ Create session and go back to previous page------
		m := make(map[string]interface{})
		m["first"] = user.First
		m["username"] = email
		m["timestamp"] = time.Now()
		this.SetSession(beeC["sessionname"], m)
		this.Redirect("/"+back, 302)
	}

	err := this.Render()
	if err != nil {
		fmt.Println(err)
	}
}

func (this *MainController) Logout() {
	beeC := conf.BeeConf(
		"sessionname",
	)

	this.activeContent("logout")
	this.DelSession(beeC["sessionname"])
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
			"vault::beego_vault_transit_key",
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
		err = trans.Create(beeC["beego_vault_transit_key"], &vault.TransitCreateOptions{})
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
		enc, err := trans.Encrypt(beeC["beego_vault_transit_key"], &vault.TransitEncryptOptions{
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

// verification email after user registered
func sendVerification(authusr *models.AuthUser, uid string, conf map[string]string) bool {
	// Step 1: -------------------- Prepare html---------------------------------
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

	flash := beego.NewFlash()

	beeC := conf.BeeConf(
		"sessionname",
		"db::beego_db_alias",
		"vault::beego_vault_address",
		"vault::beego_vault_token",
		"vault::beego_vault_transit_key",
	)

	//////////////////////////////
	// This page requires login //
	//////////////////////////////
	sess := this.GetSession(beeC["sessionname"])
	if sess == nil {
		this.Redirect("/user/login/home", 302)
		return
	}
	m := sess.(map[string]interface{})

	// Step 1:---------- Read current password hash from database-----------------
	d := &VaultDecryptVal{}
	o := orm.NewOrm()
	o.Using(beeC["beego_db_alias"])
	user := &models.AuthUser{Email: m["username"].(string)}
	switch err := o.Read(user, "Email"); {
	case err == orm.ErrNoRows:
		log.Printf("ERROR [*] No result found.. %v", err)
		flash.Error("No such user/email")
		flash.Store(&this.Controller)
		return
	case err == orm.ErrMissPK:
		log.Printf("ERROR [*] No primary key found.. %v", err)
		flash.Error("No such user/email")
		flash.Store(&this.Controller)
		return
	case err != nil:
		log.Printf("ERROR [*] Something else went wrong.. %v", err)
		flash.Error("No such user/email")
		flash.Store(&this.Controller)
		return
	case err == nil:
		// Verify() will remove uuid from user, hence if it still exists
		// it indicates that account verification (email) has not been
		// completed
		if user.Reg_key != "" {
			flash.Error("Account not verified")
			flash.Store(&this.Controller)
			return
		}
		d.vaultv1 = user.Password
		if err := d.Set(beeC); err != nil {
			log.Printf("ERROR [*] VaultDecryptVal.Set().. %v", err)
		}

		this.Data["First"] = user.First
		this.Data["Last"] = user.Last
		this.Data["Email"] = user.Email
	}

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
		password := this.GetString("password")
		password2 := this.GetString("password2")
		valid := validation.Validation{}
		valid.Required(first, "first")
		valid.Email(email, "email")
		valid.Required(current, "current")
		if valid.HasErrors() {
			log.Printf("ERROR [*] .. probably password missing..")
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
		}
		// Step 2:---------- Compare submitted password with database---------
		// Ensure that controller drops out if current password does not match
		// with DB
		if current != d.value {
			flash.Error("Bad current password")
			flash.Store(&this.Controller)
			return
		}

		// Step 3: --------- update user info in db---------------------------
		user.First = first
		user.Last = last
		user.Email = email

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

	beeC := conf.BeeConf(
		"sessionname",
		"db::beego_db_alias",
	)

	//////////////////////////////
	// This page requires login //
	//////////////////////////////
	sess := this.GetSession(beeC["sessionname"])
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
		o.Using(beeC["beego_db_alias"])
		user := &models.AuthUser{Email: m["username"].(string)}
		err := o.Read(user, "Email")
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
