package conf

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/beego/beego/v2/core/config"
)

// TODO: Prepend and separate full standard Beego config map {beego:{.},custom:{.}}
// prettyP return current config to stdout
func PrettyConf(m map[string]string) {
	jString, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		log.Printf("WARNING [*] PrettyConf.. %s", err)
	}
	log.Println("DEBUG [*] Load config.. \n" + string(jString))
}

func BeeConf(f string, confParam ...string) map[string]string {
	if f == "" {
		f = "conf/app.conf"
		if os.Getenv("BEEGO_RUNMODE") != "" {
			f = "conf/" + os.Getenv("BEEGO_RUNMODE") + ".app.conf"
		}
	}

	// initilize parser object (enables better testing capability)
	cnf, err := config.NewConfig("ini", filepath.Join(f))
	if err != nil {
		log.Fatalf("[FATAL] initialize config parser from %s.. %s", f, err)
	}

	C := make(map[string]string)
	for _, cp := range confParam {
		s := strings.Split(cp, "::")
		if len(s) > 1 { // custom configs
			cs := cnf.DefaultString(cp, "empty")
			C[s[1]] = cs
		} else { // standard config
			cs, err := cnf.String(s[0])
			if err != nil {
				fmt.Printf("PANIC [*] BeeConf.. %v", err)
				os.Exit(1)
			}
			C[s[0]] = cs
		}
	}

	return C
}
