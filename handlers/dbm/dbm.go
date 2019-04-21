package dbm

import (
	"os"
	"strings"

	"github.com/astaxie/beego/orm"
	//"github.com/kubeedge/beehive/pkg/common/config"
	//"github.com/kubeedge/beehive/pkg/common/log"
	//Blank import to run only the init function
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

const (
	// defaultDriverName is sqlite3
	defaultDriverName = "sqlite3"
	// defaultDbName is default
	defaultDbName = "default"
	// defaultDataSource is edge.db
	defaultDataSource = "bot.db"
)

var (
	driverName string
	dbName     string
	dataSource string
)

//DBAccess is Ormer object interface for all transaction processing and switching database
var DBAccess orm.Ormer

//var val orm.Params//RegisterModel registers the defined model in the orm if model is enabled
func RegisterModel(moduleName string, m interface{}) {
	//	if moduleName == "merge" || moduleName =="issue" {
	orm.RegisterModel(m)
	//	}

}

func init() {
	//Init DB info

	if driverName == "" {
		driverName = defaultDriverName
	}
	if dbName == "" {
		dbName = defaultDbName
	}
	if dataSource == "" {
		dataSource = defaultDataSource
	}

	if err := orm.RegisterDriver(driverName, orm.DRSqlite); err != nil {
		fmt.Printf("Failed to register driver: %v", err)
	}
	if err := orm.RegisterDataBase(dbName, driverName, dataSource); err != nil {
		fmt.Printf("Failed to register db: %v", err)
	}
}

//InitDBManager initialises the database by syncing the database schema and creating orm
func InitDBManager() {
	// sync database schema
	orm.RunSyncdb(dbName, false, true)

	// create orm
	DBAccess = orm.NewOrm()
	DBAccess.Using(dbName)
}

// Cleanup cleans up resources
func Cleanup() {
	cleanDBFile(dataSource)
}

// cleanDBFile removes db file
func cleanDBFile(fileName string) {
	// Remove db file
	err := os.Remove(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("DB file %s is not existing", fileName)
		} else {
			fmt.Printf("Failed to remove DB file %s: %v", fileName, err)
		}
	}
}

//func isModuleEnabled(m string) bool {
//
//	if modules != nil {
//		for _, value := range modules.([]interface{}) {
//			if m == value.(string) {
//				return true
//			}
//		}
//	}
//	return false
//}

// IsNonUniqueNameError tests if the error returned by sqlite is unique.
// It will check various sqlite versions.
func IsNonUniqueNameError(err error) bool {
	str := err.Error()
	if strings.HasSuffix(str, "are not unique") || strings.Contains(str, "UNIQUE constraint failed") || strings.HasSuffix(str, "constraint failed") {
		return true
	}
	return false
}
