//go:build mage
// +build mage

// Collection of helper scripts. Use mage -v <target> for verbose output.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"

	crypt "beego-auth/pkg/vaultv1"
)

const (
	packageName  = "github.com/olmax99/beego-auth"
)

var Aliases = map[string]interface{}{
	"e":  Encrypt,
	"d":  Decrypt,
	"sc": SyncConf,
	"t": Check,
}

var testPackages = [4]string{
	"conf",
	"models",
	"pkg/sendgridv1",
	"pkg/vaultv1",
}

// use as sources for syncing configs
var config_sources = [5]string{
	"conf/app.conf",
	"conf/dev.app.conf",
	".beego.env",
	".sendgrid_api.key",
	".vault_root_dev.key",
}

// Only with Build
// func flagEnv() map[string]string {
// 	hash, _ := sh.Output("git", "rev-parse", "--short", "HEAD")
// 	return map[string]string{
// 		"PACKAGE":     packageName,
// 		"COMMIT_HASH": hash,
// 		"BUILD_DATE":  time.Now().Format("2006-01-02T15:04:05Z0700"),
// 	}
// }

// Build binary
// func Build() error {
// 	return runWith(flagEnv(), "go", "build", "-ldflags", ldflags, buildFlags(), "-tags", buildTags(), packageName)
// }

// Check Run tests and linters
func Check() {
	mg.Deps(Vet)
	// don't run two tests in parallel, they saturate the CPUs anyway, and running two
	// causes memory issues in CI.
	mg.Deps(Test)
}

// Test Run only tests
func Test() error {
	for _, pkg := range testPackages {
		switch pkg {
		case "models":
			sql, err := sh.Output("./beego-auth", "orm", "sqlall", "-db", "auth")
			if err != nil {
				return err
			}
			if err := ioutil.WriteFile("./"+pkg+"/testdb.sql", []byte(sql), 0755); err != nil {
				return err
			}
			fallthrough
		default:
			// if err := sh.Run("go", "test", "-coverprofile="+cover, "-covermode=count", pkg); err != nil {
			// 	return err
			// }
			env := map[string]string{"GOFLAGS": testGoFlags()}
			if err:= runCmd(env, "go", "test", "./"+pkg, "-tags", buildTags(), "-v"); err != nil {
				return err
			}
		}
	}
	return nil
}

// Vet Run go vet linter
func Vet() error {
	if err := sh.Run("go", "vet", "./..."); err != nil {
		return fmt.Errorf("error running go vet: %v", err)
	}
	return nil
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


// ---------------------------- HELPERS------------------------------------

func runCmd(env map[string]string, cmd string, args ...interface{}) error {
	if mg.Verbose() {
		return runWith(env, cmd, args...)
	}
	output, err := sh.OutputWith(env, cmd, argsToStrings(args...)...)
	if err != nil {
		fmt.Fprint(os.Stderr, output)
	}

	return err
}

func runWith(env map[string]string, cmd string, inArgs ...interface{}) error {
	s := argsToStrings(inArgs...)
	return sh.RunWith(env, cmd, s...)
}

func buildTags() string {
	if envtags := os.Getenv("BEEGO_BUILD_TAGS"); envtags != "" {
		return envtags
	}
	return "none"
}

func argsToStrings(v ...interface{}) []string {
	var args []string
	for _, arg := range v {
		switch v := arg.(type) {
		case string:
			if v != "" {
				args = append(args, v)
			}
		case []string:
			if v != nil {
				args = append(args, v...)
			}
		default:
			panic("invalid type")
		}
	}

	return args
}

func isCI() bool {
	return os.Getenv("CI") != ""
}

func testGoFlags() string {
	if isCI() {
		return ""
	}
	// skips long-running test cases (not implemented, yet)
	return "-short"
}
