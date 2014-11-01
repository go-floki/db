package db

import (
	"github.com/astaxie/beego/orm"
	"github.com/go-floki/floki"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strconv"
)

func init() {
	floki.RegisterAppEventHandler("ConfigureAppEnd", func(f *floki.Floki) {
		log := f.Logger()

		f.Config.Map("datasources").EachMap(func(dsName string, values floki.ConfigMap) {
			url := values.Str("url", "")
			driver := values.Str("driver", "mysql")

			log.Println("Register datasource:", dsName, driver, url)

			orm.RegisterDataBase(dsName, driver, url, 30)

			update := values.Str("update", "none")

			if update != "none" {
				force := false

				if update == "create" {
					force = true
				}

				err := orm.RunSyncdb(dsName, force, false) //this is to create/drop tables
				if err != nil {
					log.Println(err)
					log.Println("Error syncing database")
				}

			}
		})
	})
}

func mergeParams(s string, params []interface{}) string {
	str := ""
	paramId := 0

	for _, c := range s {
		if c == '?' {
			param := params[paramId]
			switch param.(type) {
			case string:
				str += "'" + param.(string) + "'"
			case int32, int:
				str += strconv.Itoa(param.(int))
			case int64:
				str += strconv.Itoa(int(param.(int64)))
			default:
				log.Println("uknown type:", param)
			}

			paramId++

		} else {
			str += string(c)
			continue
		}
	}

	return str
}
