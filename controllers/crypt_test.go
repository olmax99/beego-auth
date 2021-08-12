package controllers_test

import (
	"beego-auth/controllers"
	"log"
	"os"
	"testing"
)

// setup Transit custom object NewCrypter()
func setup() *controllers.VaultCrypter {
	beeC := make(map[string]string)
	beeC["beego_vault_address"] = os.Getenv("VAULT_ADDR")
	beeC["beego_vault_transit_key"] = os.Getenv("BEEGO_VAULT_TRANSIT_KEY")
	beeC["beego_vault_token"] = os.Getenv("VAULT_DEV_ROOT_TOKEN")

	c := controllers.NewCrypter(beeC)
	return c
}

// requires successful Vault connection in setup
// expected: encrypt string equals decrypt string
func TestTransit(t *testing.T) {
	cases := []struct {
		a   string
		exp string
	}{
		{
			"tellmeaboutit",
			"tellmeaboutit",
		},
	}
	for _, tt := range cases {
		c := setup()
		c.SetValue(tt.a)
		if err := c.En(); err != nil {
			t.Fatalf("c.En(), %s", err)
		} else {
			log.Printf("[INFO] c.En(): %s -> %s", tt.a, c.GetVaultv1())
		}
		if err := c.De(); err != nil {
			t.Fatalf("c.De(), %s", err)
		}
		res := c.GetValue()
		if res != tt.exp {
			t.Fatalf("res %s, exp %s", tt.a, tt.exp)
		}
	}

}
