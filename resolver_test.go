package dbresolver

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
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

func TestDBResolver(t *testing.T) {
	t.Run("test queries passed to Raw", func(t *testing.T) {
		t.Run("select query", func(t *testing.T) {
			masterDB, _ := initDatabase(t)
			replicaDB, replicaDBMock := initDatabase(t)

			_ = Register(DBConfig{
				Master:   masterDB,
				Replicas: []*gorm.DB{replicaDB},
			})

			query := `SELECT * FROM users`
			replicaDBMock.ExpectQuery(regexp.QuoteMeta(query)) //.WillReturnRows(sqlmock.NewRows(nil))

			// db.Raw(query)
			replicaDB.Raw(query)

			// require.NoError(t, masterDBMock.ExpectationsWereMet())
			require.NoError(t, replicaDBMock.ExpectationsWereMet())
		})

		t.Run("insert query", func(t *testing.T) {
			masterDB, masterDBMock := initDatabase(t)
			replicaDB, replicaDBMock := initDatabase(t)

			db := Register(DBConfig{
				Master:   masterDB,
				Replicas: []*gorm.DB{replicaDB},
			})

			query := `INSERT INTO users(id, email) VALUES(1, 'abc@efg.hij')`
			masterDBMock.ExpectExec(regexp.QuoteMeta(query))

			db.Exec(query)
			require.NoError(t, replicaDBMock.ExpectationsWereMet())
			require.NoError(t, masterDBMock.ExpectationsWereMet())
		})
	})
}

func initDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "should not have failed to  init mock db")

	dB, err := gorm.Open("mysql", db)
	require.NoError(t, err, "should not have failed to open gorm")

	return dB, mock
}
