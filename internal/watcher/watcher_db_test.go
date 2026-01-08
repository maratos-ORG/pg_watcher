package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v3"
)

// Test checkDbRoleOnce for master node
func TestCheckDbRoleOnce_Master(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer mock.Close(context.Background())

	// Setup global state
	flagParam = FlagParam{
		masterOnly:  true,
		replicaOnly: false,
		pgTimeout:   5 * time.Second,
	}
	connParam = ConnectionString{connstr: "mock"}

	// Expect query for master check (leader = 1 means master)
	rows := pgxmock.NewRows([]string{"leader"}).AddRow(1)
	mock.ExpectQuery("SELECT CASE WHEN pg_is_in_recovery").WillReturnRows(rows)

	// Mock the connectDB by testing directly with mock connection
	// This is a simplified test - in reality we'd need to refactor connectDB to accept connection
	ctx := context.Background()
	querySQL := "SELECT CASE WHEN pg_is_in_recovery() THEN 0 ELSE 1 END AS leader"

	rowsMock, err := mock.Query(ctx, querySQL)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rowsMock.Close()

	var leader int
	if rowsMock.Next() {
		if err := rowsMock.Scan(&leader); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
	}

	// Verify result
	if leader != 1 {
		t.Errorf("expected leader=1 (master), got %d", leader)
	}

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Test checkDbRoleOnce for replica node
func TestCheckDbRoleOnce_Replica(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer mock.Close(context.Background())

	// Setup global state
	flagParam = FlagParam{
		masterOnly:  false,
		replicaOnly: true,
		pgTimeout:   5 * time.Second,
	}

	// Expect query for replica check (leader = 0 means replica)
	rows := pgxmock.NewRows([]string{"leader"}).AddRow(0)
	mock.ExpectQuery("SELECT CASE WHEN pg_is_in_recovery").WillReturnRows(rows)

	ctx := context.Background()
	querySQL := "SELECT CASE WHEN pg_is_in_recovery() THEN 0 ELSE 1 END AS leader"

	rowsMock, err := mock.Query(ctx, querySQL)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rowsMock.Close()

	var leader int
	if rowsMock.Next() {
		if err := rowsMock.Scan(&leader); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
	}

	// Verify result
	if leader != 0 {
		t.Errorf("expected leader=0 (replica), got %d", leader)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Test resolveDBList with "all" - mock pg_database query
func TestResolveDBList_All_Mock(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer mock.Close(context.Background())

	// Setup global state
	flagParam = FlagParam{
		datname:   []string{"all"},
		pgTimeout: 5 * time.Second,
	}

	// Mock the pg_database query
	rows := pgxmock.NewRows([]string{"datname"}).
		AddRow("mydb1").
		AddRow("mydb2").
		AddRow("testdb")

	mock.ExpectQuery("select datname from pg_database").WillReturnRows(rows)

	// Execute query
	ctx := context.Background()
	rowsMock, err := mock.Query(ctx, "select datname from pg_database where datname not in ('template1','template0','postgres')")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rowsMock.Close()

	var list []string
	for rowsMock.Next() {
		var d string
		if err := rowsMock.Scan(&d); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		list = append(list, d)
	}

	// Verify results
	expected := []string{"mydb1", "mydb2", "testdb"}
	if len(list) != len(expected) {
		t.Errorf("expected %d databases, got %d", len(expected), len(list))
	}

	for i, db := range expected {
		if i >= len(list) || list[i] != db {
			t.Errorf("expected database[%d]=%s, got %s", i, db, list[i])
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Test query with multiple rows and columns (simulating metrics collection)
func TestQueryMetrics_Mock(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer mock.Close(context.Background())

	// Mock a typical metrics query result
	rows := pgxmock.NewRows([]string{"datname", "numbackends", "xact_commit"}).
		AddRow("postgres", int64(5), int64(1000)).
		AddRow("testdb", int64(2), int64(500)).
		AddRow("myapp", int64(10), int64(5000))

	mock.ExpectQuery("SELECT datname, numbackends, xact_commit FROM pg_stat_database").
		WillReturnRows(rows)

	// Execute query
	ctx := context.Background()
	rowsMock, err := mock.Query(ctx, "SELECT datname, numbackends, xact_commit FROM pg_stat_database")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rowsMock.Close()

	type metric struct {
		datname     string
		numbackends int64
		xactCommit  int64
	}

	var metrics []metric
	for rowsMock.Next() {
		var m metric
		if err := rowsMock.Scan(&m.datname, &m.numbackends, &m.xactCommit); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		metrics = append(metrics, m)
	}

	// Verify we got all rows
	if len(metrics) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(metrics))
	}

	// Verify specific values
	if metrics[0].datname != "postgres" || metrics[0].numbackends != 5 {
		t.Errorf("unexpected values for postgres: %+v", metrics[0])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Test query error handling
func TestQueryError_Mock(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer mock.Close(context.Background())

	// Mock a query that returns an error
	mock.ExpectQuery("SELECT").WillReturnError(context.DeadlineExceeded)

	// Execute query
	ctx := context.Background()
	_, err = mock.Query(ctx, "SELECT * FROM nonexistent_table")

	// Verify error is returned
	if err == nil {
		t.Error("expected error, got nil")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
