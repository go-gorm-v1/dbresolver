<!--
  Title: DBResolver
  Description: Resolve between read and write database automatically for gorm. go-gorm-v1 ,dbresolver
  Author: amitavaghosh1
  -->

# DBResolver

DBResolver for [gorm v1](https://v1.gorm.io/docs/index.html). This adds functionality to switch between read and write databases.

## Quick Start

```go
import (
  "github.com/go-gorm-v1/dbresolver"
  "github.com/jinzhu/gorm"
  _ "github.com/mattn/go-sqlite3"
)

func Setup() *dbresolver.Database {
    masterDB, err := gorm.Open("sqlite3", "./testdbs/users_write.db")
    if err != nil {
      log.Fatal("failed to connect to db", err)
    }

    replicaDBs := []*gorm.DB{}
    
    replica, err := gorm.Open("sqlite3", "./testdbs/users_read_a.db")
    if err != nil {
      log.Fatal("failed to connect to db", err)
    }

    replicaDBs = append(replicaDBs, replica)

    replica, err = gorm.Open("sqlite3", "./testdbs/users_read_b.db")
    if err != nil {
      log.Fatal("failed to connect to db", err)
    }


    replicaDBs = append(replicaDBs, replica)

    return dbresolver.Register(dbresolver.DBConfig{
        Master:   masterDB,
        Replicas: replicaDBs,
        // By default we have
        // Policy: &dbresolver.RoundRobalancer{}
        // Other option
        // Policy: &dbresolver.RandomBalancer{}
    })
}

db := Setup()

db.Raw(`SELECT * FROM users`)
```


### Transaction

It is possible to provide the option to use read or write forcefully.

```go
// Raw By default uses read db
db.Raw(`SELECT * FROM users`)

// Use write db
db.WithMode(dbresolver.DBWriteMode).Raw(`SELECT * FROM users`)

// Use read db
db.WithMode(dbresolver.DBReadMode).Exec(`DELETE FROM users`)
```


### Load Balancing

By default we have two balancers

```go
// RandomBalancer
dbresolver.DBConfig{
    Policy: &dbresolver.RoundRobalancer{}
}

// RandomBalancer
dbresolver.DBConfig{
    Policy: &dbresolver.RandomBalancer{}
}
```

You can provide your own load balancer. The `Balancer` interface is defined as such

```go
type Balancer interface {
    Get() int64
}

```

