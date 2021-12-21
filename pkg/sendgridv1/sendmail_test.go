package sendgridv1_test

import (
	"beego-auth/pkg/sendgridv1"
	"log"
	"os"
	"testing"
)

// setup check sendgrid apikey is set
func setup() string {
	key := os.Getenv("BEEGO_SENDGRID_API_KEY")
	if len(key) < 1 {
		log.Fatal("ERROR [-] missing sendgrid api key.")
	}
	return key
}

func TestSgGetV3WhitelabelDomains(t *testing.T) {
	k := setup()
	sg_req := sendgridv1.NewSgReq(k)

	_, err := sg_req.SgGet(string("/v3/whitelabel/domains"))
	if err != nil {
		log.Fatalf("DEBUG [*] SgGetV3WhitelabelDomains: .. %s", err)
	}
	sg_req.ConfirmSgAccountDomain()

}

func TestSgSendMailVerification(t *testing.T) {
	test_mail_addr := os.Getenv("BEEGO_TEST_EMAIL")
	k := setup()
	sg_mail := sendgridv1.NewSgSendMail(
		k, os.Getenv("BEEGO_SENDGRID_OWN_SUPPORT"), "",
		sendgridv1.MailTo("Carl", test_mail_addr),
		sendgridv1.PrepareHtml("Selma", "http://localhost:5115/console", "verification.tpl"),
	)
	res := sg_mail.Send("just a test")
	if res.StatusCode != 202 {
		log.Fatalf("DEBUG [*] SgSendMailVerification.. %#v", res)
	}
}
