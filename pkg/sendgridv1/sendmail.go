package sendgridv1

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

var (
	abs_path = os.Getenv("BEEGO_ABS_PATH")
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

// TODO: Interface that reflects any Api Response struct
type sgRequestGet struct {
	res    []sgApiResponse
	apikey string
	url    string
	host   string
}

type sgReqConf func(wdr *sgRequestGet)

// NewSgReq Handler for general purpose sendgrid GET requests
func NewSgReq(apikey string, conf ...sgReqConf) sgRequestGet {
	wld := sgRequestGet{
		res:    []sgApiResponse{},
		apikey: apikey,
		host:   "https://api.sendgrid.com",
	}
	for _, c := range conf {
		c(&wld)
	}
	return wld
}

type sgSendConf func(sm *sgSendMail)

type sgSendMail struct {
	client  sendgrid.Client
	from    mail.Email
	to      mail.Email
	resp    rest.Response
	content string
}

func NewSgSendMail(apikey, beego_own_support, send_name string, conf ...sgSendConf) sgSendMail {
	name := send_name
	if send_name == "" {
		name = strings.Split(beego_own_support, "@")[0]
	}
	sm := sgSendMail{
		client: *sendgrid.NewSendClient(apikey),
		from: mail.Email{
			Name:    name,
			Address: beego_own_support,
		},
		resp: rest.Response{},
	}
	for _, c := range conf {
		c(&sm)
	}
	return sm
}

// mailTo attach address of mail receiver to new sgSendMail.to variable
func MailTo(to_name, to_addr string) sgSendConf {
	return func(sm *sgSendMail) {
		sm.to = mail.Email{
			Name:    to_name,
			Address: to_addr,
		}
	}
}

func PrepareHtml(user_name, link, template_name string) sgSendConf {
	t, err := template.New(template_name).ParseFiles(filepath.Join(abs_path, "pkg/sendgridv1/templates", template_name))
	if err != nil {
		log.Printf("ERROR [*] Parse html failed.. %v", err)
	}
	data := struct {
		User string
		Link string
	}{
		User: user_name,
		Link: link,
	}

	var tplOut bytes.Buffer
	if err := t.Execute(&tplOut, data); err != nil {
		log.Printf("ERROR [*] PrepareHtml: tpl.exec failed.. %v", err)
	}
	return func(sm *sgSendMail) {
		sm.content = tplOut.String()
	}
}

func PrepareHtmlWithReset(user_name, link, reset_pass, template_name string) sgSendConf {
	t, err := template.New(template_name).ParseFiles(filepath.Join(abs_path, "pkg/sendgridv1/templates", template_name))
	if err != nil {
		log.Printf("ERROR [*] Parse html failed.. %v", err)
	}

	data := struct {
		User string
		Link string
		RanB string
	}{
		User: user_name,
		Link: link,
		RanB: reset_pass,
	}

	var tplOut bytes.Buffer
	if err := t.Execute(&tplOut, data); err != nil {
		log.Printf("ERROR [*] PrepareHtmlWithReset: tpl.exec failed.. %v", err)
	}
	return func(sm *sgSendMail) {
		sm.content = tplOut.String()
	}
}

func (sgReq *sgRequestGet) SgGet(url string) ([]sgApiResponse, error) {
	accReq := sendgrid.GetRequest(sgReq.apikey, url, sgReq.host)
	accResp, err := sendgrid.MakeRequestRetryWithContext(context.TODO(), accReq)
	if err != nil {
		log.Printf("WARNING [-] SendgridApi.. Please verify that the api key corresponds to the sg account.")
		return nil, err
	}
	err = json.Unmarshal([]byte(accResp.Body), &sgReq.res)
	if err != nil {
		log.Printf("WARNING [-] SendgridApi response, %s", err)
		return nil, err
	}
	return sgReq.res, nil
}

// ConfirmSgAccountDomain compare account domain to sendgrid support sender address
func (sgReq *sgRequestGet) ConfirmSgAccountDomain() {
	exp := strings.Split(os.Getenv("BEEGO_SENDGRID_OWN_SUPPORT"), "@")
	if exp[1] != sgReq.res[0].Domain {
		log.Fatalf("PANIC [-] Sendgrid Sender Email '%s'.. ensure that the sender email is matching with the authenticated user domain.", exp)
	}
}

func (sgSend *sgSendMail) Send(subject string) *rest.Response {
	message := mail.NewSingleEmail(&sgSend.from, subject, &sgSend.to, "", sgSend.content)
	response, err := sgSend.client.Send(message)
	if err != nil {
		log.Printf("ERROR [*] sgSendMail.Send.. %v", err)
		return nil
	}
	return response
}
