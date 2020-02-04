// Copyright 2019-2020 Kosc Telecom.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dispatcher

import (
	"database/sql"
	"fmt"
	"horus/log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var (
	db                       *sqlx.DB
	lockDevStmt              *sql.Stmt
	unlockDevStmt            *sql.Stmt
	unlockAllDevStmt         *sql.Stmt
	unlockDevFromReportStmt  *sql.Stmt
	unlockFromOngoingStmt    *sql.Stmt
	unlockFromAgentStmt      *sql.Stmt
	setDevLastPolledAt       *sql.Stmt
	setDevLastPingedAt       *sql.Stmt
	insertMetricLastPolledAt *sql.Stmt
	insertReportStmt         *sql.Stmt
	updReportStmt            *sql.Stmt
	checkAgentStmt           *sql.Stmt
)

// InitDB initializes db connection and prepares the db statements
func InitDB(dsn string) error {
	var err error

	log.Debug2f("opening db connection to %q", dsn)
	db, err = sqlx.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("connect db: %v", err)
	}
	lockDevStmt, err = db.Prepare(`UPDATE devices
                                      SET is_polling = true
                                    WHERE id = $1`)
	if err != nil {
		return fmt.Errorf("prepare lockDevStmt: %v", err)
	}
	unlockDevStmt, err = db.Prepare(`UPDATE devices
                                        SET is_polling = false
                                      WHERE id = $1`)
	if err != nil {
		return fmt.Errorf("prepare unlockDevStmt: %v", err)
	}
	unlockAllDevStmt, err = db.Prepare(`UPDATE devices
                                           SET is_polling = false
                                         WHERE last_polled_at < NOW() - ($1::TEXT || ' seconds')::INTERVAL
                                           AND is_polling = true`)
	if err != nil {
		return fmt.Errorf("prepare unlockAllDevStmt: %v", err)
	}
	unlockDevFromReportStmt, err = db.Prepare(`UPDATE devices
                                                  SET is_polling = false
                                                WHERE id = (SELECT device_id
                                                              FROM reports
                                                             WHERE uuid = $1)`)
	unlockFromAgentStmt, err = db.Prepare(`UPDATE devices
                                              SET is_polling = false
                                            WHERE id IN (SELECT device_id
                                                           FROM reports
                                                          WHERE agent_id = $1
                                                            AND report_received_at IS NULL
                                                            AND requested_at >= NOW() - INTERVAL '15 minutes')`)
	if err != nil {
		return fmt.Errorf("prepare unlockDevFromAgentStmt: %v", err)
	}
	unlockFromOngoingStmt, err = db.Prepare(`UPDATE devices
                                                SET is_polling = false
                                              WHERE last_polled_at < NOW() - (polling_frequency::TEXT || ' seconds')::INTERVAL
                                                AND id NOT IN (SELECT device_id
                                                                 FROM reports
                                                                WHERE uuid = ANY($1))`)
	if err != nil {
		return fmt.Errorf("prepare unlockFromOngoingStmt: %v", err)
	}
	setDevLastPolledAt, err = db.Prepare(`UPDATE devices
                                             SET last_polled_at = NOW()
                                           WHERE id = $1`)
	if err != nil {
		return fmt.Errorf("prepare setLastPollDate: %v", err)
	}
	setDevLastPingedAt, err = db.Prepare(`UPDATE devices
                                             SET last_pinged_at = NOW()
                                           WHERE id = ANY($1)`)
	if err != nil {
		return fmt.Errorf("prepare setLastPingDate: %v", err)
	}
	insertMetricLastPolledAt, err = db.Prepare(`INSERT INTO metric_poll_times
                                                            (device_id, metric_id, last_polled_at)
                                                     VALUES ($1, $2, NOW())
                                                ON CONFLICT (device_id, metric_id)
                                                  DO UPDATE
                                                        SET last_polled_at = NOW()`)
	if err != nil {
		return fmt.Errorf("prepare insertMetricLastPolledAt: %v", err)
	}
	insertReportStmt, err = db.Prepare(`INSERT INTO reports
                                                    (uuid, device_id, agent_id, post_status, requested_at)
                                             VALUES ($1, $2, $3, $4, NOW())`)
	if err != nil {
		return fmt.Errorf("prepare insertReportStmt: %v", err)
	}
	updReportStmt, err = db.Prepare(`UPDATE reports
                                        SET report_received_at = NOW(),
                                            poll_duration_ms = $2,
                                            poll_error = $3
                                      WHERE uuid = $1`)
	if err != nil {
		return fmt.Errorf("prepare updReportStmt: %v", err)
	}
	checkAgentStmt, err = db.Prepare(`UPDATE agents
                                         SET last_checked_at = NOW(),
                                             is_alive = $2,
                                             load = $3
                                       WHERE id = $1`)
	if err != nil {
		return fmt.Errorf("prepare checkAgentStmt: %v", err)
	}
	return nil
}

// ReleaseDB closes the db connection.
func ReleaseDB() {
	db.Close()
}

// sqlExec executes the prepared statement stmt with its args,
// and logs and returns the db error if any
func sqlExec(id interface{}, reqName string, stmt *sql.Stmt, args ...interface{}) error {
	log.Debug3f("%v - sql exec %s", id, reqName)
	_, err := stmt.Exec(args...)
	if err != nil {
		log.Errorf("sql exec %s (%v): %v", reqName, id, err)
	}
	return err
}
