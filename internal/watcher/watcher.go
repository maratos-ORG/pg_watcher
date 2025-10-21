package watcher

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/semaphore"
)

type FlagParam struct {
	sqlQuery        []string
	labelColumnsArr []string
	ignoredColumns  map[string]bool
	SQLSpliter      string
	masterOnly      bool
	replicaOnly     bool
	datname         []string
	prefixMetric    string
	jobs            int
	pgTimeout       time.Duration
}

type ConnectionString struct {
	connstr string
}

var (
	flagParam FlagParam
	connParam ConnectionString
)

// Run is the former main(): it executes the full program flow.
// All fatal exits are replaced by returning errors. The outer main()
// decides how to exit.
func Run(ctxParent context.Context, fp *FlagParam, cp *ConnectionString) error {
	// Keep original globals to avoid refactoring the rest of the code.
	flagParam = *fp
	connParam = *cp

	// 1) database list
	dbList, err := resolveDBList(ctxParent)
	if err != nil {
		return err
	}

	// 2) role check (if requested)
	if flagParam.masterOnly || flagParam.replicaOnly {
		if err := checkDbRoleOnce(ctxParent); err != nil {
			return err
		}
	}

	// 3) parallel processing limited by -j
	sem := semaphore.NewWeighted(int64(flagParam.jobs))
	for _, name := range dbList {
		if err := sem.Acquire(ctxParent, 1); err != nil {
			return fmt.Errorf("failed to acquire semaphore: %v", err)
		}
		go func(dbname string) {
			defer sem.Release(1)
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[db=%s] panic recovered: %v", dbname, r)
				}
			}()
			if err := processDB(ctxParent, dbname); err != nil {
				log.Printf("DB %s: %v\n", dbname, err)
			}
		}(name)
	}
	// wait for all goroutines to finish
	if err := sem.Acquire(ctxParent, int64(flagParam.jobs)); err != nil {
		return fmt.Errorf("final acquire: %v", err)
	}
	return nil
}

func resolveDBList(ctxParent context.Context) ([]string, error) {
	// if len(flagParam.datname) > 0 && strings.ToLower(flagParam.datname[0]) == "all" {
	if len(flagParam.datname) > 0 && strings.EqualFold(flagParam.datname[0], "all") {
		conn, cancelConn, err := connectDB(ctxParent, "postgres")
		if err != nil {
			return nil, err
		}
		defer cancelConn()
		defer closeConn(ctxParent, conn)

		rows, cancelQ, err := queryWithTimeout(ctxParent, conn,
			"select datname from pg_database where datname not in ('template1','template0','postgres')")
		if err != nil {
			return nil, err
		}
		defer cancelQ()
		defer rows.Close()

		var list []string
		for rows.Next() {
			var d string
			if err := rows.Scan(&d); err != nil {
				return nil, err
			}
			list = append(list, d)
		}
		return list, rows.Err()
	}
	return flagParam.datname, nil
}

// connectDB: timeout applies only to establishing the connection
func connectDB(ctxParent context.Context, dbname string) (*pgx.Conn, context.CancelFunc, error) {
	if dbname == "" {
		dbname = "postgres"
	}
	ctxConn, cancelConn := context.WithTimeout(ctxParent, flagParam.pgTimeout)
	conn, err := pgx.Connect(ctxConn, connParam.connstr+" dbname="+dbname)
	if err != nil {
		cancelConn()
		return nil, nil, err
	}
	return conn, cancelConn, nil
}

// queryWithTimeout: per-query timeout
func queryWithTimeout(ctxParent context.Context, conn *pgx.Conn, sql string) (pgx.Rows, context.CancelFunc, error) {
	ctxQ, cancelQ := context.WithTimeout(ctxParent, flagParam.pgTimeout)
	rows, err := conn.Query(ctxQ, sql)
	if err != nil {
		cancelQ()
		return nil, nil, err
	}
	return rows, cancelQ, nil
}

// checkDbRoleOnce: verifies node role if master-only / replica-only is requested
func checkDbRoleOnce(ctxParent context.Context) error {
	conn, cancelConn, err := connectDB(ctxParent, "postgres")
	if err != nil {
		return err
	}
	defer cancelConn()
	defer closeConn(ctxParent, conn)

	rows, cancelQ, err := queryWithTimeout(ctxParent, conn,
		"SELECT CASE WHEN pg_is_in_recovery() THEN 0 ELSE 1 END AS leader")
	if err != nil {
		return err
	}
	defer cancelQ()
	defer rows.Close()

	var leader int
	if rows.Next() {
		if err := rows.Scan(&leader); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if leader == 0 && flagParam.masterOnly {
		return errors.New("INFO: --master-only requested but node is replica")
	}
	if leader == 1 && flagParam.replicaOnly {
		return errors.New("INFO: --replica-only requested but node is master")
	}
	return nil
}

// processDB: main metrics collection logic
func processDB(parentCtx context.Context, dbname string) error {
	conn, cancelConn, err := connectDB(parentCtx, dbname)
	if err != nil {
		return err
	}
	defer cancelConn()
	defer closeConn(parentCtx, conn)

	for _, sqlText := range flagParam.sqlQuery {
		if err := func(sqlText string) error {
			rows, cancelQ, err := queryWithTimeout(parentCtx, conn, sqlText)
			if err != nil {
				return fmt.Errorf("query error: %w", err)
			}
			defer cancelQ()
			defer rows.Close()

			fds := rows.FieldDescriptions()

			// precompute per-column metadata (iterate in fds order)
			forced := makeForcedLabelsSet(flagParam.labelColumnsArr)
			type colMeta struct {
				idx     int
				name    string
				ignored bool
				forced  bool
				label   string // normalized label name
				metric  string // normalized metric name
			}
			metas := make([]colMeta, 0, len(fds))
			for i, fd := range fds {
				name := fd.Name
				ignored := false
				if flagParam.ignoredColumns != nil {
					_, ignored = flagParam.ignoredColumns[name]
				}
				metas = append(metas, colMeta{
					idx:     i,
					name:    name,
					ignored: ignored,
					forced:  forced[name],
					label:   normalizeName(name),
					metric:  normalizeName(fmt.Sprintf("%s_%s", flagParam.prefixMetric, name)),
				})
			}

			for rows.Next() {
				vals, err := rows.Values()
				if err != nil {
					return fmt.Errorf("scan values: %w", err)
				}
				// safety guard: values must match field count
				if len(vals) != len(fds) {
					log.Printf("[db=%s] row mismatch: vals=%d fds=%d", dbname, len(vals), len(fds))
					continue
				}

				var lb strings.Builder
				first := true

				type mval struct {
					name string
					val  float64
				}
				metricsBuf := make([]mval, 0, len(metas))

				// single pass over columns in SELECT order
				for _, m := range metas {
					if m.ignored {
						continue
					}
					if m.idx < 0 || m.idx >= len(vals) {
						continue
					}
					v := vals[m.idx]

					// labels: forced columns are always labels; otherwise strings become labels
					if m.forced {
						if !first {
							lb.WriteByte(',')
						}
						fmt.Fprintf(&lb, `%s="%s"`, m.label, labelVal(v))
						first = false
						continue // do not duplicate as metric
					}
					switch v.(type) {
					case string, []byte:
						if !first {
							lb.WriteByte(',')
						}
						fmt.Fprintf(&lb, `%s="%s"`, m.label, labelVal(v))
						first = false
						continue
					}

					// metrics: only numeric
					if f, ok := toFloat64(v); ok && !math.IsNaN(f) && !math.IsInf(f, 0) {
						metricsBuf = append(metricsBuf, mval{name: m.metric, val: f})
					}
				}

				labels := lb.String()

				// print all metrics with the prepared label set
				for _, it := range metricsBuf {
					if labels != "" {
						fmt.Printf("%s{%s,db=%q} %g\n", it.name, labels, dbname, it.val)
					} else {
						fmt.Printf("%s{db=%q} %g\n", it.name, dbname, it.val)
					}
				}
			}
			return rows.Err()
		}(sqlText); err != nil {
			return err
		}
	}
	return nil
}

// ParseFlags is your former processingFlag() but:
// 1) accepts `build` to print on -version;
// 2) returns FlagParam and ConnectionString to avoid fatal exits in library code.
func ParseFlags(build string) (*FlagParam, *ConnectionString) {
	version := flag.Bool("version", false, "print current version")
	connPtr := flag.String("conn", "user=postgres host=127.0.0.1 port=5435", "PostgreSQL conn string (libpq format)")
	pgTimeout := flag.Duration("pg-timeout", 5*time.Second, "Global timeout for PostgreSQL operations (connect + query)")
	dbnamePtr := flag.String("db-name", "", "DB name(s): 'all' or comma-separated list")
	sqlPtr := flag.String("sql-cmd", "", "SQL query text")
	sqlfilePtr := flag.String("sql-file", "", "File with SQL command(s)")
	labelsPtr := flag.String("labels", "", "Label columns (comma-separated). If not specified, all string columns will be used as labels.")
	ignoredColumnsPtr := flag.String("ignoredColumns", "", "Columns to exclude (comma-separated)")
	SQLSpliter := flag.String("SQLSpliter", "", "Delimiter for splitting multiple SQL commands")
	masterOnlyPtr := flag.Bool("master-only", false, "Execute only on master")
	replicaOnlyPtr := flag.Bool("replica-only", false, "Execute only on replica")
	prefixMetric := flag.String("prefixMetric", "pgwatch", "Metric prefix")
	jobsPtr := flag.Int("j", 1, "Max concurrent databases to process")

	flag.Parse()

	if *version {
		fmt.Println(build)
		os.Exit(0)
	}
	if *dbnamePtr == "" {
		log.Fatalln("ERROR: -db-name must be specified (use 'all' or list).")
	}
	flagParam.datname = strings.Split(*dbnamePtr, ",")
	connParam.connstr = *connPtr
	flagParam.pgTimeout = *pgTimeout

	if *labelsPtr != "" {
		for _, it := range strings.Split(*labelsPtr, ",") {
			flagParam.labelColumnsArr = append(flagParam.labelColumnsArr, strings.TrimSpace(it))
		}
	}
	if *ignoredColumnsPtr != "" {
		flagParam.ignoredColumns = make(map[string]bool)
		for _, it := range strings.Split(*ignoredColumnsPtr, ",") {
			flagParam.ignoredColumns[strings.TrimSpace(it)] = true
		}
	}

	if (*sqlPtr == "" && *sqlfilePtr == "") || (*sqlPtr != "" && *sqlfilePtr != "") {
		log.Fatalln("ERROR: use either -sql-cmd or -sql-file (exactly one).")
	}
	if *sqlPtr != "" {
		if *SQLSpliter != "" {
			flagParam.sqlQuery = strings.Split(strings.TrimRight(*sqlPtr, ";"), *SQLSpliter)
		} else {
			flagParam.sqlQuery = append(flagParam.sqlQuery, *sqlPtr)
		}
	}
	if *sqlfilePtr != "" {
		content, err := os.ReadFile(*sqlfilePtr)
		if err != nil {
			log.Fatal(err)
		}
		sqlText := string(content)
		if *SQLSpliter != "" {
			flagParam.sqlQuery = strings.Split(strings.TrimRight(sqlText, ";"), *SQLSpliter)
		} else {
			flagParam.sqlQuery = append(flagParam.sqlQuery, sqlText)
		}
	}
	if *SQLSpliter != "" {
		flagParam.SQLSpliter = *SQLSpliter
	}

	flagParam.masterOnly = *masterOnlyPtr
	flagParam.replicaOnly = *replicaOnlyPtr
	if *prefixMetric != "" {
		flagParam.prefixMetric = *prefixMetric
	}
	if *jobsPtr <= 0 {
		*jobsPtr = 1
	}
	flagParam.jobs = *jobsPtr

	return &flagParam, &connParam
}

// closeConn closes connection with its own timeout
func closeConn(ctxParent context.Context, c *pgx.Conn) {
	ctx, cancel := context.WithTimeout(ctxParent, flagParam.pgTimeout)
	defer cancel()
	_ = c.Close(ctx)
}

// toFloat64 converts most numeric-like values to float64
func toFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case nil:
		return 0, false
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case int32:
		return float64(x), true
	case int16:
		return float64(x), true
	case int8:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint64:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint8:
		return float64(x), true
	case json.Number:
		if f, err := x.Float64(); err == nil {
			return f, true
		}
	case []byte:
		if f, err := strconv.ParseFloat(string(x), 64); err == nil {
			return f, true
		}
	case string:
		if f, err := strconv.ParseFloat(x, 64); err == nil {
			return f, true
		}
	default:
		s := fmt.Sprint(v)
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, true
		}
		type float64Valuer interface{ Float64Value() (float64, bool) }
		if fv, ok := v.(float64Valuer); ok {
			return fv.Float64Value()
		}
	}
	return 0, false
}

// normalizeName converts arbitrary column/metric names into Prometheus-friendly identifiers
func normalizeName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, " ", "_")
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	// if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
	if s != "" && s[0] >= '0' && s[0] <= '9' {
		s = "_" + s
	}
	return s
}

// makeForcedLabelsSet builds a set from -labels CSV
func makeForcedLabelsSet(arr []string) map[string]bool {
	m := make(map[string]bool, len(arr))
	for _, s := range arr {
		s = strings.TrimSpace(s)
		if s != "" {
			m[s] = true
		}
	}
	return m
}

// labelVal formats label values safely
func labelVal(v any) string {
	switch x := v.(type) {
	case []byte:
		return string(x)
	default:
		return fmt.Sprint(v)
	}
}
