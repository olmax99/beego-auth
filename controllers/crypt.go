package controllers

import (
	"log"
	"sync"

	vault "github.com/olmax99/vaultgo"
)

// Container for retrieving decrypted strings via Vault Transit
type VaultCrypter struct {
	vaultv1 string
	value   string
	client  *vault.Transit
	config  map[string]string
	lock    sync.RWMutex
}

// Create the decrypted user password value from the vault:v1 string
func (vc *VaultCrypter) De() error {
	vc.lock.Lock()
	defer vc.lock.Unlock()
	v, err := vc.client.Decrypt(vc.config["beego_vault_transit_key"], &vault.TransitDecryptOptions{
		Ciphertext: vc.vaultv1,
	})
	if err != nil {
		log.Printf("ERROR [*] Decrypt failed.. %v", err)
		return err
	}
	vc.value = v.Data.Plaintext
	return nil
}

// Send plaintext to Vault and derive the vault:v1 value
func (vc *VaultCrypter) En() error {
	vc.lock.Lock()
	defer vc.lock.Unlock()
	enc, err := vc.client.Encrypt(vc.config["beego_vault_transit_key"], &vault.TransitEncryptOptions{
		Plaintext: vc.value,
	})
	if err != nil {
		log.Printf("ERROR [*] Encrypt failed.. %v", err)
		return err
	}
	vc.vaultv1 = enc.Data.Ciphertext
	return nil
}

// Create a new Vault Transit container
func NewCrypter(beeCfg map[string]string) *VaultCrypter {
	c := &VaultCrypter{}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.config = beeCfg
	client, err := vault.NewClient(c.config["beego_vault_address"], vault.WithCaPath(""))
	if err != nil {
		log.Println("PANIC [-] Could not connect with Vault")
		return nil
	}
	if t := client.Token(); t == "" {
		log.Println("WARNING [*] No token found in environment. Try config..")
		client.SetToken(c.config["beego_vault_token"])
	}
	c.client = client.Transit()
	err = c.client.Create(c.config["beego_vault_transit_key"], &vault.TransitCreateOptions{})
	if err != nil {
		log.Printf("ERROR [*] Could not create Vault client.. %v", err)
		return nil
	}
	return c
}
