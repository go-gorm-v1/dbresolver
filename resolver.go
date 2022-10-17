package dbresolver

import (
	"log"
	"strings"

	"github.com/jinzhu/gorm"
)

type DBConfig struct {
	Master   *gorm.DB
	Replicas []*gorm.DB
	Policy   Balancer
}

type DbActionMode string

var (
	DbWriteMode DbActionMode = "write"
	DbReadMode               = "read"
)

type Database struct {
	*gorm.DB // this makes sure, calls like Save Update etc, goes through default source db
	Config   DBConfig
}

func Register(config DBConfig) *Database {
	if config.Master == nil {
		log.Fatal("config.Master db cannot be nil")
	}

	var balancer Balancer = NewRoundRobalancer(len(config.Replicas))

	switch config.Policy.(type) {
	case *RandomBalancer:
		balancer = NewRoundRobalancer(len(config.Replicas))
	case *RoundRobalancer:
		balancer = NewRandomBalancer(len(config.Replicas))
	default:
		balancer = NewRoundRobalancer(len(config.Replicas))
	}

	config.Policy = balancer
	return &Database{DB: config.Master, Config: config}
}

func (d *Database) WithMode(dbMode DbActionMode) *gorm.DB {
	if dbMode == DbWriteMode {
		return d.getMaster()
	}

	return d.getReplica()
}

func (d *Database) Exec(sql string, values ...interface{}) *gorm.DB {
	master := d.getMaster()

	if !isDML(strings.ToLower(sql)) {
		replica := d.getReplica()
		return replica.Exec(sql, values...)
	}

	return master.Exec(sql, values...)
}

func (d *Database) Raw(sql string, values ...interface{}) *gorm.DB {
	replica := d.getReplica()
	master := d.DB

	if isDML(strings.ToLower(sql)) {
		return master.Raw(sql, values...)
	}

	return replica.Raw(sql, values...)
}

func (d *Database) Where(query interface{}, args ...interface{}) *gorm.DB {
	replica := d.getReplica()
	return replica.Where(query, args...)
}

/*
func (d *Database) Find(query interface{}, args ...interface{}) *gorm.DB {
	return d.getReplica().Find(query, args...)
}

func (d *Database) First(query interface{}, args ...interface{}) *gorm.DB {
	return d.getReplica().First(query, args...)
}

func (d *Database) Last(query interface{}, args ...interface{}) *gorm.DB {
	return d.getReplica().Last(query, args...)
}

func (d *Database) Take(query interface{}, args ...interface{}) *gorm.DB {
	replica := d.getReplica()
	return replica.Take(query, args...)
}

func (d *Database) Save(value interface{}) *gorm.DB {
	return d.getMaster().Save(value)
}
*/

func (d *Database) getReplica() *gorm.DB {
	nextIdx := d.Config.Policy.Get()

	if len(d.Config.Replicas) > 0 && nextIdx < int64(len(d.Config.Replicas)) {
		return d.Config.Replicas[nextIdx]
	}

	return d.DB
}

func (d *Database) getMaster() *gorm.DB {
	return d.DB
}

func isDML(sql string) bool {
	sql = strings.ToLower(strings.TrimSpace(sql))

	isSelect := len(sql) > 7 && strings.EqualFold(sql[:6], "select")
	isLockQuery := strings.Contains(sql[6:], "for update") ||
		strings.Contains(sql[6:], "for share")

	if isSelect && isLockQuery {
		return true
	}
	if isSelect {
		return false
	}

	return true
}

func getDbModeManuallyIfPresent(values ...interface{}) *DbActionMode {
	if len(values) == 0 {
		return nil
	}

	lastValue := values[len(values)-1]

	mode, ok := lastValue.(DbActionMode)
	if !ok {
		return nil
	}
	return &mode
}

/*
dbMode := getDbModeManuallyIfPresent(values...)
if dbMode == nil || string(*dbMode) == DbReadMode {
	log.Println("using replica")
	return replica.Raw(sql, values...)
}
*/
