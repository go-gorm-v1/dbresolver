package dbresolver

import (
	"errors"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIsDML(t *testing.T) {
	t.Run("valid queries", func(t *testing.T) {
		createQuery := "CREATE TABLE users()"
		require.Equal(t, isDML(createQuery), true)

		selectQuery := "SELECT COUNT(1) FROM users"
		require.Equal(t, isDML(selectQuery), false)

		insertQuery := "INSERT INTO users VALUES()"
		require.Equal(t, isDML(insertQuery), true)

		updateQuery := `UPDATE users SET email="abcd@gmail.com"`
		require.Equal(t, isDML(updateQuery), true)

		deleteQuery := "DELETE FROM users"
		require.Equal(t, isDML(deleteQuery), true)

		selectShareQuery := "SELECT * FROM FOR UPDATE"
		require.Equal(t, isDML(selectShareQuery), true)

		selectShareQuery = "SELECT * FROM FOR SHARE"
		require.Equal(t, isDML(selectShareQuery), true)
	})

	t.Run("invalid queries, defaults to modification query, to allow select masteer db", func(t *testing.T) {
		invalidQuery := "FOOBAR"
		require.Equal(t, isDML(invalidQuery), true)
	})
}

type DBResolverSuite struct {
	suite.Suite
	database *Database
	MasterDB *gorm.DB
	Replicas []*gorm.DB
}

func (dbs *DBResolverSuite) SetupTest() {
	masterDB, err := gorm.Open("sqlite3", "./testdbs/users_write.db")
	t := dbs.Suite.T()
	require.NoError(t, err, "failed to connect to write db")

	replicaDBs := []*gorm.DB{}
	replica, err := gorm.Open("sqlite3", "./testdbs/users_read_a.db?mode=ro")
	require.NoError(t, err, "failed  to connect to read instance a")

	replicaDBs = append(replicaDBs, replica)

	replica, err = gorm.Open("sqlite3", "./testdbs/users_read_b.db?mode=ro")
	require.NoError(t, err, "failed  to connect to read instance b")

	replicaDBs = append(replicaDBs, replica)

	dbs.database = Register(DBConfig{
		Master:   masterDB,
		Replicas: replicaDBs,
	})
}

type User struct {
	ID    string
	Email string
}

func (dbs *DBResolverSuite) TestRawQueries() {
	t := dbs.Suite.T()

	t.Run("with automatic switching to replica", func(t *testing.T) {
		recv := make(chan string, 2)

		for i := 0; i < 2; i++ {
			go func() {
				var result User
				err := dbs.database.Raw("SELECT email FROM users WHERE id=?", "a").Scan(&result).Error
				require.NoError(t, err, "should not have failed to get users from database")

				recv <- result.Email
			}()
		}

		emails := []string{}

		var fetchFailed error

		for i := 0; i < 2; i++ {
			select {
			case email := <-recv:
				emails = append(emails, email)
			case <-time.After(2 * time.Second):
				fetchFailed = errors.New("failed")
			}
		}

		require.NoError(t, fetchFailed, "should not have failed to get data")
		require.ElementsMatch(t, emails, []string{"dad@gmail.com", "foo@gmail.com"})
	})

	t.Run("selecting db mode manually", func(t *testing.T) {
		var result User

		err := dbs.database.WithMode(DbWriteMode).Raw("SELECT * FROM users WHERE id = ?", "a").Scan(&result).Error
		require.NoError(t, err, "should not have failed to get users from write db")

		require.Equal(t, result.Email, "baz@gmail.com")
	})
}

func (dbs *DBResolverSuite) TestExecQuery() {
	t := dbs.Suite.T()

	err := dbs.database.Exec(`INSERT INTO users (id, email) VALUES(?, ?)`, "c", "boo@gmail.com").Error
	require.NoError(t, err, "failed to insert into db")

	var result User

	err = dbs.database.WithMode(DbWriteMode).Raw("SELECT email FROM users WHERE id = ?", "c").Scan(&result).Error
	require.NoError(t, err, "failed to get email from id")

	require.Equal(t, result.Email, "boo@gmail.com")

	err = dbs.database.Exec("DELETE FROM users WHERE id = ?", "c").Error
	require.NoError(t, err, "failed to delete from db")
}

func (dbs *DBResolverSuite) TestUpdate() {
	t := dbs.Suite.T()

	err := dbs.database.Exec(`INSERT INTO users (id, email) VALUES(?, ?)`, "c", "boo@gmail.com").Error
	require.NoError(t, err, "failed to insert into db")

	var result User

	err = dbs.database.Model(&User{}).Where("id = ?", "c").Update("email", "boohoo@gmail.com").Error
	require.NoError(t, err, "should not have failed to update")

	err = dbs.database.WithMode(DbWriteMode).Raw("SELECT email FROM users WHERE id = ?", "c").Scan(&result).Error
	require.NoError(t, err, "should not have failed to get from database")

	require.Equal(t, result.Email, "boohoo@gmail.com")

	result = User{}

	err = dbs.database.Raw("SELECT email FROM users WHERE id = ?", "c").Scan(&result).Error
	require.Error(t, err, "should have failed to get from database")

	err = dbs.database.Exec("DELETE FROM users WHERE id = ?", "c").Error
	require.NoError(t, err, "failed to delete from db")
}

func TestDBResolver(t *testing.T) {
	suite.Run(t, new(DBResolverSuite))
}
