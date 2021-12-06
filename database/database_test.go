package database

import (
	"context"
	"os"
	"testing"
)

const dbName = "dbutils"

func TestMain(m *testing.M) {
	code := m.Run()

	os.Remove(dbName + ".db")
	os.Exit(code)
}

func TestCreateSqliteConnection(t *testing.T) {
	testcases := []struct {
		Name          string
		DBSettings    DatabaseConfig
		ExpectedError error
	}{
		{
			Name: "Should create connection",
			DBSettings: DatabaseConfig{
				Engine: "sqlite",
				Name:   dbName,
			},
			ExpectedError: nil,
		},
		{
			Name: "Should return error",
			DBSettings: DatabaseConfig{
				Engine: "sqlite",
				Name:   "",
			},
			ExpectedError: ErrInvalidDBName,
		},
	}

	ctx := context.Background()

	for i := range testcases {
		tc := testcases[i]

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			db, err := CreateSqliteConnection(ctx, tc.DBSettings)
			if err != tc.ExpectedError {
				t.Errorf("\nExpected: %v\nReceived: %v\n", tc.ExpectedError, err)
			}

			if db != nil {
				db.Close()
			}
		})
	}

}
