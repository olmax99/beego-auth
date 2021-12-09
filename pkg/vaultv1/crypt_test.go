package vaultv1_test

import (
	"log"
	"os"
	"testing"

	crypt "beego-auth/pkg/vaultv1"

	vault "github.com/olmax99/vaultgo"
)

// setup Transit custom object NewCrypter()
func setup() *crypt.VaultCrypter {
	beeC := make(map[string]string)
	beeC["beego_vault_address"] = os.Getenv("VAULT_ADDR")
	beeC["beego_vault_transit_key"] = os.Getenv("BEEGO_VAULT_TRANSIT_KEY")
	beeC["beego_vault_token"] = os.Getenv("VAULT_DEV_ROOT_TOKEN")

	c := crypt.NewCrypter(beeC)

	return c
}

func setup2() string {
	beeC := make(map[string]string)
	beeC["beego_vault_address"] = os.Getenv("VAULT_ADDR")
	beeC["beego_vault_transit_key"] = os.Getenv("BEEGO_VAULT_TRANSIT_KEY")
	beeC["beego_vault_token"] = os.Getenv("VAULT_DEV_ROOT_TOKEN")

	c := crypt.NewCrypter(beeC)
	ct := c.GetTransitClient()

	v, err := ct.Encrypt(beeC["beego_vault_transit_key"], &vault.TransitEncryptOptions{
		Plaintext: "tellmeaboutit",
	})
	if err != nil {
		log.Printf("ERROR [*] Decrypt failed.. %v", err)
	}

	return v.Data.Ciphertext
}

// requires successful Vault connection in setup and setValue performs encryption
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
		log.Printf("DEBUG [+] next tt.a='%s'", tt.a)
		if err := c.SetValue(tt.a); err != nil {
			t.Fatalf("c.en(), %s", err)
		}
	}

}

// TestDecryption ... returns plain text string
func TestDecryption(t *testing.T) {
	c := setup()
	ct := c.GetTransitClient()
	encrypted := setup2()
	v, err := ct.Decrypt(os.Getenv("BEEGO_VAULT_TRANSIT_KEY"), &vault.TransitDecryptOptions{
		Ciphertext: encrypted,
	})
	if err != nil {
		log.Printf("ERROR [*] Decrypt failed.. %v", err)
	}
	log.Printf("DEBUG [+] Transit.Decrypt -> '%s'", v.Data.Plaintext)
	if v.Data.Plaintext == "" {
		t.Fatalf("Decrypt did not return a value.")
	}
}

// TestDecryptWithSetVaultv1 .. decrypts a previously encrypted plaintext string
func TestDecryptWithSetVaultv1(t *testing.T) {
	c := setup()
	ciphertext_random := setup2()
	if err := c.SetVaultv1(ciphertext_random); err != nil {
		t.Fatalf("%s", err)
	}
	if !c.Match("tellmeaboutit") {
		t.Fatalf("act '%s'", c.GetValue())
	}
}
