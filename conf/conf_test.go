// DONE Beego is automatically trying to load the config. But it is not clear how to change
//  this behavior:
//  - Without the file existing BEFORE running the test, it will fail
// Answer:
//  - The culprit lies in beego/beego/v2/server/web/config.go init()
//  - Use beego/beego/v2/core/config for creating a new config parser
// ---------------------------------------------------------------------------------------
// 2021/11/30 02:01:41.886 [W]  init global config instance failed. If you donot use this,
// just ignore it.  open conf/app.conf: no such file or directory

package conf_test

import (
	"beego-auth/conf"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func setup(tdir string) {
	// create testfile
	file, err := os.Create(filepath.Join(tdir, "test.app.conf"))
	if err != nil {
		fmt.Printf("[Error] os.Create %s/test.app.conf", tdir)
		os.Exit(1)
	}

	if _, err = io.WriteString(file, "httpport=5151\n\n[db]\nbeego_db_alias=default"); err != nil {
		file.Close()
	}
}

func TestBeeConfStringConvertion(t *testing.T) {
	tmpDir := t.TempDir()
	setup(tmpDir)

	// test case
	cases := []struct {
		a, b string
		exp  map[string]string
	}{
		{
			"httpport",
			"db::beego_db_alias",
			map[string]string{"httpport": "5151", "beego_db_alias": "default"},
		},
	}
	for _, tt := range cases {
		res := conf.BeeConf(filepath.Join(tmpDir, "test.app.conf"), "httpport", "db::beego_db_alias")
		if res["httpport"] != tt.exp["httpport"] {
			t.Fatalf("res %s, exp %s", res["httpport"], tt.exp["httpport"])
		}
		if res["beego_db_alias"] != tt.exp["beego_db_alias"] {
			t.Fatalf("res %s, exp %s", res["beego_db_alias"], tt.exp["beego_db_alias"])
		}
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			fmt.Printf("[INFO] clean %s", tmpDir)
		}
	})
}
