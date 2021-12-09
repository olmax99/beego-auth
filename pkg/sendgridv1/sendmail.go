package sendgridv1

import (
	"beego-auth/models"
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type Dkim1 struct {
	Valid bool   `json:"valid"`
	Type  string `json:"type"`
	Host  string `json:"host"`
	Data  string `json:"data"`
}

type Dkim2 struct {
	Valid bool   `json:"valid"`
	Type  string `json:"type"`
	Host  string `json:"host"`
	Data  string `json:"data"`
}

type MailCname struct {
	Valid bool   `json:"valid"`
	Type  string `json:"type"`
	Host  string `json:"host"`
	Data  string `json:"data"`
}

type DNS struct {
	MailCname MailCname `json:"mail_cname"`
	Dkim1     Dkim1     `json:"dkim1"`
	Dkim2     Dkim2     `json:"dkim2"`
}

// Response ../whitelabel/domains
// https://docs.sendgrid.com/api-reference/domain-authentication/list-all-authenticated-domains
type sgApiResponse struct {
	ID                      int           `json:"id"`
	UserID                  int           `json:"user_id"`
	Subdomain               string        `json:"subdomain"`
	Domain                  string        `json:"domain"`
	Username                string        `json:"username"`
	Ips                     []interface{} `json:"ips"`
	CustomSpf               bool          `json:"custom_spf"`
	Default                 bool          `json:"default"`
	Legacy                  bool          `json:"legacy"`
	AutomaticSecurity       bool          `json:"automatic_security"`
	Valid                   bool          `json:"valid"`
	DNS                     DNS           `json:"dns"`
	LastValidationAttemptAt int           `json:"last_validation_attempt_at"`
}

// get_Domain() retrieves the Domain from a 'whitelabel/domains' response
func (r *sgApiResponse) get_Domain() string {
	return r.Domain
}

// confirmDomainAuthenticated verify domain is verified in SG account
func (r *sgApiResponse) confirmDomainAuthenticated(c map[string]string) {
	accReq := sendgrid.GetRequest(c["beego_sg_api_key"], "/v3/whitelabel/domains", "")
	accResp, err := sendgrid.MakeRequestRetryWithContext(context.TODO(), accReq)
	if err != nil {
		log.Printf("WARNING [-] SendgridApi.. Please verify that the api key corresponds to the sg account.")
	}
	var ag []sgApiResponse
	err = json.Unmarshal([]byte(accResp.Body), &ag)
	if err != nil {
		log.Printf("WARNING [-] SendgridApi response, %s", err)
	}

	if dc := strings.Split(os.Getenv("BEEGO_SENDGRID_OWN_SUPPORT"), "@"); dc[1] != ag[0].get_Domain() {
		log.Fatalf("PANIC [-] Sendgrid Sender Email '%s'.. ensure that the sender email is matching with the authenticated user domain.", c["beego_sg_own_support"])
	}
}

// confirmSgApiKey client check, verify SG api confirmSgApiKey
func confirmSgApiKey(c map[string]string) {
	authReq := sendgrid.GetRequest(c["beego_sg_api_key"], "/v3/templates", "")
	authResp, err := sendgrid.MakeRequestRetryWithContext(context.TODO(), authReq)
	if err != nil {
		log.Printf("WARNING [-] SendgridClient.. Please verify that the api key corresponds to the sg account.")
	}
	switch authResp.StatusCode {
	case 200:
	case 403:
		log.Fatalf("PANIC [-] SendgridClient.. %d, wrong api key.", authResp.StatusCode)
	default:
		log.Fatalf("PANIC [-] SendgridClient.. %d", authResp.StatusCode)
	}
}

// verification email after user registered
func SendVerification(authusr *models.AuthUser, uid string, conf map[string]string) bool {
	// Step 1: -------------------- Prepare html---------------------------------
	link := "http://localhost:" + conf["httpport"] + "/user/verify/" + uid
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	t, err := template.New("verification.tpl").ParseFiles(pwd + "/pkg/sendgridv1/verification.tpl")
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
	}
	switch response.StatusCode {
	case 200:
		return true
	case 202:
		return true
	case 403:
		log.Printf("DEBUG [*] Sendgrid 403, body: %s", response.Body)
		return false
	default:
		log.Printf("DEBUG [*] Sendgrid response, status: %v", response.StatusCode)
		log.Printf("DEBUG [*] Sendgrid response, body: %v", response.Body)
		log.Printf("DEBUG [*] Sendgrid response, header: %#v", response.Headers)
		return false
	}
}

// verification email after user cancelled
func SendCancellation(authusr *models.AuthUser, uid string, conf map[string]string) bool {
	// Step 1: -------------------- Prepare html---------------------------------
	link := "http://localhost:" + conf["httpport"] + "/user/cancel/" + uid
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	t, err := template.New("cancellation.tpl").ParseFiles(pwd + "/pkg/sendgridv1/cancellation.tpl")
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
		log.Printf("DEBUG [*] Sendgrid response, status: %v", response.StatusCode)
		log.Printf("DEBUG [*] Sendgrid response, body: %v", response.Body)
		log.Printf("DEBUG [*] Sendgrid response, header: %#v", response.Headers)
	}
	return true
}

// verification email after user initiated to reset password
func SendReset(authusr *models.AuthUser, uid string, pass []byte, conf map[string]string) bool {
	// Step 1: -------------------- Prepare html---------------------------------
	link := "http://localhost:" + conf["httpport"] + "/user/genpass1/" + uid
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	t, err := template.New("reset.tpl").ParseFiles(pwd + "/pkg/sendgridv1/reset.tpl")
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
		log.Printf("DEBUG [*] Sendgrid response, status: %v", response.StatusCode)
		log.Printf("DEBUG [*] Sendgrid response, body: %v", response.Body)
		log.Printf("DEBUG [*] Sendgrid response, header: %#v", response.Headers)
	}
	return true
}

// Confirm sendgrid api key and is valid and authenticated domain matches with sender email
func Confirm(c map[string]string) {
	ag := sgApiResponse{}
	confirmSgApiKey(c)
	ag.confirmDomainAuthenticated(c)
}
