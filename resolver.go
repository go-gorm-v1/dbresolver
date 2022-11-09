package dbresolver

import (
	"log"
	"strings"

	"github.com/go-gorm-v1/dbresolver/hooks"
	"github.com/jinzhu/gorm"
)

type DBConfig struct {
	Master      *gorm.DB
	Replicas    []*gorm.DB
	Policy      Balancer
	DefaultMode *DbActionMode
}

type DbActionMode string

var (
	DbWriteMode DbActionMode = "write"
	DbReadMode  DbActionMode = "read"
)

var (
	EventBeforeDBSelect string = "before::select_db"
	EventAfterDBSelect  string = "after:select_db"
	EventBeforeQueryRun string = "before::query_run"
)

type Database struct {
	// this makes sure, calls like Save Update etc, goes through default source db
	*gorm.DB
	Config DBConfig
	Hooks  *hooks.EventStore
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

	if config.DefaultMode == nil {
		config.DefaultMode = &DbReadMode
	}

	return &Database{
		DB:     config.Master,
		Config: config,
		Hooks:  hooks.NewEventStore(),
	}
}

func (d *Database) WithMode(dbMode DbActionMode) *gorm.DB {
	dbConfig := d.Config
	dbConfig.DefaultMode = &dbMode

	nd := &Database{
		DB:     d.DB,
		Config: dbConfig,
		Hooks:  d.Hooks,
	}

	return nd.selectSource()
	// if dbMode == DbWriteMode {
	// return d.getMaster()
	// }

	// return d.getReplica()
}

func (d *Database) Exec(sql string, values ...interface{}) *gorm.DB {
	master := d.getMaster()

	if !isDML(strings.ToLower(sql)) {
		return d.getReplica().Exec(sql, values...)
	}

	d.Hooks.Emit(EventBeforeQueryRun, sql, values)
	return master.Exec(sql, values...)
}

func (d *Database) Raw(sql string, values ...interface{}) *gorm.DB {
	if isDML(strings.ToLower(sql)) {
		return d.getMaster().Raw(sql, values...)
	}

	d.Hooks.Emit(EventBeforeQueryRun, sql, values)
	return d.getReplica().Raw(sql, values...)
}

func (d *Database) Where(query interface{}, args ...interface{}) *gorm.DB {
	d.Hooks.Emit(EventBeforeQueryRun, query, args)
	return d.selectSource().Where(query, args...)
}

func (d *Database) Find(query interface{}, args ...interface{}) *gorm.DB {
	d.Hooks.Emit(EventBeforeQueryRun, query, args)
	return d.selectSource().Find(query, args...)
}

func (d *Database) First(query interface{}, args ...interface{}) *gorm.DB {
	d.Hooks.Emit(EventBeforeQueryRun, query, args)
	return d.selectSource().First(query, args...)
}

func (d *Database) Last(query interface{}, args ...interface{}) *gorm.DB {
	d.Hooks.Emit(EventBeforeQueryRun, query, args)
	return d.selectSource().Last(query, args...)
}

func (d *Database) Take(query interface{}, args ...interface{}) *gorm.DB {
	d.Hooks.Emit(EventBeforeQueryRun, query, args)
	return d.selectSource().Take(query, args...)
}

func (d *Database) Count(value interface{}) *gorm.DB {
	return d.selectSource().Count(value)
}

func (d *Database) Save(value interface{}) *gorm.DB {
	return d.getMaster().Save(value)
}

func (d *Database) getReplica() (db *gorm.DB) {
	nextIdx := d.Config.Policy.Get()

	db = d.DB

	if len(d.Config.Replicas) > 0 && nextIdx < int64(len(d.Config.Replicas)) {
		db = d.Config.Replicas[nextIdx]
	}

	d.Hooks.Emit(EventAfterDBSelect, "replica", db, nextIdx)
	return
}

func (d *Database) getMaster() *gorm.DB {
	d.Hooks.Emit(EventAfterDBSelect, "master", d.DB, 0)
	return d.DB
}

func (d *Database) selectSource() *gorm.DB {
	d.Hooks.Emit(EventBeforeDBSelect, *d.Config.DefaultMode)

	if DbWriteMode == *d.Config.DefaultMode {
		return d.getMaster()
	}

	return d.getReplica()
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
