//go:build mage
// +build mage

// Collection of helper scripts.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"

	crypt "beego-auth/pkg/vaultv1"
)

var Aliases = map[string]interface{}{
	"e":  Encrypt,
	"d":  Decrypt,
	"sc": SyncConf,
}

// use as sources for syncing configs
var config_sources = []string{
	"conf/app.conf",
	"conf/dev.app.conf",
	".beego.env",
	".sendgrid_api.key",
	".vault_root_dev.key",
}

// SyncConf backup config files (non-git) to target folder
func SyncConf() error {
	dst := os.Getenv("BEEGO_CONFIG_PATH")
	if len(dst) == 0 {
		errConfPath := errors.New("Ensure that BEEGO_CONFIG_PATH is defined.")
		return errConfPath
	}
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return err
	}
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	sources := make([]string, len(config_sources))
	for i := range config_sources {
		sources[i] = filepath.Join(path, config_sources[i])
	}
	// reports wheather any source file is newer than its counterpart in target
	newer, err := target.Dir(dst, sources...)
	if err != nil {
		return err
	}
	if newer {
		for _, cs := range config_sources {
			tgt := filepath.Join(dst, filepath.Base(cs))
			src := filepath.Join(path, cs)
			cerr := sh.Copy(tgt, src)
			if cerr != nil {
				return cerr
			}
		}
		if err := sh.RunV("ls", "-al", dst); err != nil {
			return err
		}
		return nil
	}
	return nil
}

// Decrypt decrypt data previously encrypted by Vault Transit
func Decrypt(ctx context.Context, enc string) error {
	if !strings.HasPrefix(enc, "vault:v1:") {
		errPrefix := errors.New("Vault Transit strings usually start with 'vault:v1:***'.")
		return errPrefix
	}
	m := make(map[string]string)
	m["beego_vault_address"] = os.Getenv("VAULT_ADDR")
	m["beego_vault_transit_key"] = os.Getenv("BEEGO_VAULT_TRANSIT_KEY")
	m["beego_vault_token"] = os.Getenv("VAULT_DEV_ROOT_TOKEN")
	crypt := crypt.NewCrypter(m)
	if err := crypt.SetVaultv1(enc); err != nil {
		return err
	}
	fmt.Printf("------\n'%s'\n", crypt.GetValue())
	return nil
}

// Encrypt encrypt data using the Vault Transit Api
func Encrypt(ctx context.Context, plain string) error {
	m := make(map[string]string)
	m["beego_vault_address"] = os.Getenv("VAULT_ADDR")
	m["beego_vault_transit_key"] = os.Getenv("BEEGO_VAULT_TRANSIT_KEY")
	m["beego_vault_token"] = os.Getenv("VAULT_DEV_ROOT_TOKEN")
	crypt := crypt.NewCrypter(m)
	if err := crypt.SetValue(plain); err != nil {
		return err
	}
	fmt.Printf("------\n'%s'\n", crypt.GetVaultv1())
	return nil
}
