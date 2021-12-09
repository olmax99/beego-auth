package vaultv1

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

// GetTransit returns initial *vault.Transit client object
func (vc *VaultCrypter) GetTransitClient() *vault.Transit {
	return vc.client
}

// GetClient shows Vault mount point
func (vc *VaultCrypter) GetClient() string {
	return vc.client.MountPoint
}

// GetValue returns private plaintext value
func (vc *VaultCrypter) GetValue() string {
	return vc.value
}

// GetVaultv1 returns private var vaultv1
func (vc *VaultCrypter) GetVaultv1() string {
	return vc.vaultv1
}

// Vaultv1 Set private var vaultv1
func (vc *VaultCrypter) SetVaultv1(v string) error {
	vc.vaultv1 = v
	if err := vc.de(); err != nil {
		log.Printf("ERROR [*] VaultDecrypter.Set().. %v", err)
		return err
	}
	return nil
}

// Value Set private var value
func (vc *VaultCrypter) SetValue(v string) error {
	vc.value = v
	if err := vc.en(); err != nil {
		log.Printf("ERROR [*] VaultCrypter.En().. %v", err)
		return err
	}
	return nil
}

func (vc *VaultCrypter) Match(p string) bool {
	return vc.value == p
}

func (vc *VaultCrypter) FromBytes(b []byte) error {
	vc.value = string(b)
	if err := vc.en(); err != nil {
		log.Printf("ERROR [*] From bytes failed.. %v", err)
		return err
	}
	return nil
}

// Create the decrypted user password value from the vault:v1 string
func (vc *VaultCrypter) de() error {
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
func (vc *VaultCrypter) en() error {
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

// Confirm Transit client can connect to Vault
func Confirm(c map[string]string) {
	if nc := NewCrypter(c); nc == nil {
		log.Fatalf("PANIC [-] VaultClient.. Please verify address, token, and key.")
	}
}

// Create a new Vault Transit container using Beego Configuration parameters
func NewCrypter(beeCfg map[string]string) *VaultCrypter {
	c := &VaultCrypter{}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.config = beeCfg
	client, err := vault.NewClient(c.config["beego_vault_address"], vault.WithCaPath(""))
	if err != nil {
		log.Println("WARNING [-] Could not connect with Vault")
		return nil
	}
	if t := client.Token(); t == "" {
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
