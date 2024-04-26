package agent

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestParseTask(t *testing.T) {
	var db *sql.DB
	file, _ := os.Create("data_1.db")
	file.Close()
	db, _ = sql.Open("sqlite3", "data_1.db")
	db.Exec("CREATE TABLE IF NOT EXISTS Operations (operation TEXT, duration INTEGER, user_id INTEGER)")
	db.Exec("INSERT OR IGNORE INTO Operations (operation, duration, user_id) VALUES ('+', 1, 1)")
	db.Exec("INSERT OR IGNORE INTO Operations (operation, duration, user_id) VALUES ('-', 1, 1)")
	db.Exec("INSERT OR IGNORE INTO Operations (operation, duration, user_id) VALUES ('*', 1, 1)")
	db.Exec("INSERT OR IGNORE INTO Operations (operation, duration, user_id) VALUES ('/', 1, 1)")
	cases := []struct {
		name   string
		task   string
		answer float64
	}{
		{
			name:   "sum",
			task:   "1+1",
			answer: 2.0,
		},
		{
			name:   "minus",
			task:   "1-2",
			answer: -1.0,
		},
		{
			name:   "division",
			task:   "5/2",
			answer: 2.5,
		},
		{
			name:   "multiply",
			task:   "6*5",
			answer: 30.0,
		},
		{
			name:   "hardtask",
			task:   "100/(2+(2*(2*2)+2*(2-2)))",
			answer: 10.0,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := Parse_Task([]rune(tc.task), db, 1)
			if got != tc.answer {
				t.Errorf("Parse task(%v) = %v, want = %v", tc.task, got, tc.answer)
			}
		})
	}
	db.Close()
	os.Remove("data_1.db")
}
