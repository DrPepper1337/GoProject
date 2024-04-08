package main

import (
	"database/sql"
	"errors"
	"html/template"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func CheckBrackets(equation string) bool {
	brackets := 0
	for _, elem := range equation {
		if brackets < 0 {
			return false
		}
		if string(elem) == "(" {
			brackets += 1
		} else if string(elem) == ")" {
			brackets -= 1
		}
	}
	return true
}

func ValidEquation(equation string, start, end int) bool {
	if end-start <= 0 {
		return false
	}
	end = min(end, len(equation))
	seenDot := false
	for i := start; i < end; i++ {
		if equation[i] == '(' {
			seenDot = false
			validInner := false
			parenthesis := 1
			for j := i + 1; j < end; j++ {
				if equation[j] == '(' {
					parenthesis++
				} else if equation[j] == ')' {
					parenthesis--
					if parenthesis == 0 {
						if !ValidEquation(equation, i+1, j) {
							return false
						}
						i = j + 1
						validInner = true
						break
					}
				}
			}
			if !validInner {
				return false
			}
		} else if equation[i] == ')' {
			return false
		} else if isOperator(rune(equation[i])) {
			seenDot = false
			if i == start {
				if equation[i] == '*' || equation[i] == '/' {
					return false
				}
			} else {
				if isOperator(rune(equation[i-1])) {
					return false
				}
			}
			if i == end-1 || isOperator(rune(equation[i+1])) {
				return false
			}
		} else if !('0' <= equation[i] && equation[i] <= '9') {
			if equation[i] == '.' && !seenDot {
				seenDot = true
			} else {
				return false
			}
		}
	}
	return true
}

func isOperator(c rune) bool {
	return c == '+' || c == '-' || c == '*' || c == '/'
}

func sum(n1, n2 float64) float64 {
	return n1 + n2
}

func minus(n1, n2 float64) float64 {
	return n1 - n2
}

func multiply(n1, n2 float64) float64 {
	return n1 * n2
}

func division(n1, n2 float64) float64 {
	return n1 / n2
}

func OperationTime(operation string) int {
	rows, _ := db.Query("SELECT duration FROM Operations WHERE operation = ?", operation)
	defer rows.Close()

	var duration int
	if rows.Next() {
		rows.Scan(&duration)
	}
	return duration
}

func delete_useless_brackets(end_str []rune) []rune {
	brackets_number2 := 0
	if string(end_str[0]) == "(" && string(end_str[len(end_str)-1]) == ")" {
		flag := true
		for bnd, elem := range end_str {
			if bnd != len(end_str)-1 && bnd != 0 && brackets_number2 == 0 {
				flag = false
			}
			if string(elem) == "(" {
				brackets_number2 += 1
			} else if string(elem) == ")" {
				brackets_number2 -= 1
			}
		}
		if flag {
			end_str = end_str[1 : len(end_str)-1]
		}
	}
	return end_str
}

func Parse_Task(task []rune) float64 {
	state := ""
	current_string := ""
	brackets_number := 0
	if n, err := strconv.ParseFloat(string(task), 64); err == nil {
		return n
	}
	for i, letter := range task {
		if string(letter) == "(" {
			if brackets_number == 0 {
				state = "("
				brackets_number += 1
			} else {
				brackets_number += 1
			}
			current_string += string(letter)
		} else if string(letter) == "." {
			current_string += string(letter)
		} else if string(letter) == ")" {
			if brackets_number == 1 {
				brackets_number--
				state = ""
			} else {
				brackets_number--
			}
			current_string += string(letter)
		} else if letter <= 57 && 48 <= letter {
			current_string += string(letter)
		} else if string(letter) == "+" {
			if state == "(" {
				current_string += string(letter)
			} else if state == "" {
				end_str := []rune(task)[i+1:]
				var wg sync.WaitGroup
				var n1, n2 float64
				wg.Add(2)
				go func() {
					defer wg.Done()
					n1 = Parse_Task(delete_useless_brackets([]rune(current_string)))
				}()
				go func() {
					defer wg.Done()
					n2 = Parse_Task(delete_useless_brackets(end_str))
				}()
				wg.Wait()
				time.Sleep(time.Duration(OperationTime("+")) * time.Millisecond)
				return sum(n1, n2)
			}
		} else if string(letter) == "-" {
			if state == "(" || i == 0 || (task[i-1] > 57 && 48 > task[i-1]) {
				current_string += string(letter)
			} else if state == "" {
				end_str := task[i+1:]
				var wg sync.WaitGroup
				var n1, n2 float64
				wg.Add(2)
				go func() {
					defer wg.Done()
					n1 = Parse_Task(delete_useless_brackets([]rune(current_string)))
				}()
				go func() {
					defer wg.Done()
					n2 = Parse_Task(delete_useless_brackets(end_str))
				}()
				wg.Wait()
				time.Sleep(time.Duration(OperationTime("-")) * time.Millisecond)
				return minus(n1, n2)
			}
		} else if string(letter) == "*" {
			if state == "(" {
				current_string += string(letter)
			} else if state == "" {
				flag := true
				brackets_number1 := 0
				for _, symbol := range task[i+1:] {
					if string(symbol) == "(" {
						brackets_number1 += 1
					} else if string(symbol) == ")" {
						brackets_number1 -= 1
					} else if (string(symbol) == "+" || string(symbol) == "-") && brackets_number1 == 0 {
						flag = false
					}
				}
				if flag {
					end_str := task[i+1:]
					var wg sync.WaitGroup
					var n1, n2 float64
					wg.Add(2)
					go func() {
						defer wg.Done()
						n1 = Parse_Task(delete_useless_brackets([]rune(current_string)))
					}()
					go func() {
						defer wg.Done()
						n2 = Parse_Task(delete_useless_brackets(end_str))
					}()
					wg.Wait()
					time.Sleep(time.Duration(OperationTime("*")) * time.Millisecond)
					return multiply(n1, n2)
				}
				current_string += string(letter)
			}
		} else if string(letter) == "/" {
			if state == "(" {
				current_string += string(letter)
			} else if state == "" {
				flag := true
				brackets_number1 := 0
				for _, symbol := range task[i+1:] {
					if string(symbol) == "(" {
						brackets_number1 += 1
					} else if string(symbol) == ")" {
						brackets_number1 -= 1
					} else if (string(symbol) == "+" || string(symbol) == "-" || string(symbol) == "/" || string(symbol) == "*") && brackets_number1 == 0 {
						flag = false
					}
				}
				if flag {
					end_str := task[i+1:]
					var wg sync.WaitGroup
					var n1, n2 float64
					wg.Add(2)
					go func() {
						defer wg.Done()
						n1 = Parse_Task(delete_useless_brackets([]rune(current_string)))
					}()
					go func() {
						defer wg.Done()
						n2 = Parse_Task(delete_useless_brackets(end_str))
					}()
					wg.Wait()
					time.Sleep(time.Duration(OperationTime("/")) * time.Millisecond)
					return division(n1, n2)
				}
				current_string += string(letter)
			}
		}
	}
	return 1.0
}

func createTable(db *sql.DB) {
	db.Exec("CREATE TABLE IF NOT EXISTS Tasks (id INTEGER PRIMARY KEY AUTOINCREMENT, task TEXT, status TEXT, result REAL, start TEXT, finish TEXT)")
	db.Exec("CREATE TABLE IF NOT EXISTS Operations (operation TEXT PRIMARY KEY, duration INTEGER)")
	db.Exec("CREATE TABLE IF NOT EXISTS Calc (id INTEGER PRIMARY KEY AUTOINCREMENT, calc TEXT)")
	db.Exec("INSERT OR IGNORE INTO Operations (operation, duration) VALUES ('+', 1)")
	db.Exec("INSERT OR IGNORE INTO Operations (operation, duration) VALUES ('-', 1)")
	db.Exec("INSERT OR IGNORE INTO Operations (operation, duration) VALUES ('*', 1)")
	db.Exec("INSERT OR IGNORE INTO Operations (operation, duration) VALUES ('/', 1)")
}

func add_task(task string) {
	equation := strings.ReplaceAll(task, " ", "")
	equation = strings.ReplaceAll(equation, ",", ".")

	rows, _ := db.Query("SELECT * FROM Calc WHERE calc=''")
	columns, _ := rows.Columns()
	ar_of_rows := make([][]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtr := make([]interface{}, len(columns))
		for i := range columns {
			valuePtr[i] = &values[i]
		}
		rows.Scan(valuePtr...)
		ar_of_rows = append(ar_of_rows, values)
	}
	if len(ar_of_rows) == 0 {
		stmt, _ := db.Prepare("INSERT OR IGNORE INTO Tasks (task, status) VALUES (?, ?)")
		stmt.Exec(equation, "Ожидает свободного сервера для начала вычислений")
		stmt.Close()
		for len(ar_of_rows) == 0 {
			time.Sleep(200 * time.Millisecond)
			rows, _ := db.Query("SELECT * FROM Calc WHERE calc=''")
			columns, _ := rows.Columns()
			ar_of_rows = make([][]interface{}, 0)
			for rows.Next() {
				values := make([]interface{}, len(columns))
				valuePtr := make([]interface{}, len(columns))
				for i := range columns {
					valuePtr[i] = &values[i]
				}
				rows.Scan(valuePtr...)
				ar_of_rows = append(ar_of_rows, values)
			}
		}
	} else {
		stmt, _ := db.Prepare("INSERT OR IGNORE INTO Tasks (task) VALUES (?)")
		stmt.Exec(equation)
		stmt.Close()
	}
	stmt, _ := db.Prepare("UPDATE Calc SET calc=? WHERE id=?")
	stmt.Exec(equation, ar_of_rows[0][0])
	stmt.Close()
	if ValidEquation(equation, 0, len(equation)) && CheckBrackets(equation) {
		stmt, _ := db.Prepare("UPDATE Tasks SET status=?, start=? WHERE task = ?")
		defer stmt.Close()
		stmt.Exec("Проводится подсчёт выражения", time.Now().Format("01-02-2006 15:04:05"), equation)
		result := Parse_Task(delete_useless_brackets([]rune(equation)))
		if math.IsInf(result, 0) {
			stmt, _ = db.Prepare("UPDATE Tasks SET status=?, result = ?, finish = ? WHERE task = ?")
			stmt.Exec("Деление на 0", 0, time.Now().Format("01-02-2006 15:04:05"), equation)
		} else {
			stmt, _ = db.Prepare("UPDATE Tasks SET status=?, result = ?, finish = ? WHERE task = ?")
			stmt.Exec("Успешно", result, time.Now().Format("01-02-2006 15:04:05"), equation)
		}
	} else {
		stmt, _ := db.Prepare("INSERT OR IGNORE INTO Tasks (task, status, result, start, finish) VALUES (?, ?, ?, ?, ?)")
		defer stmt.Close()
		stmt.Exec(equation, "Неверный формат", 0, time.Now().Format("01-02-2006 15:04:05"), time.Now().Format("01-02-2006 15:04:05"))
	}
	stmt, _ = db.Prepare("UPDATE Calc SET calc=? WHERE id=?")
	stmt.Exec("", ar_of_rows[0][0])
	stmt.Close()
}

func add_task_page(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()
		var task = r.FormValue("task")
		go add_task(task)
	}
	tmpl, _ := template.ParseFiles("templates\\base.html", "templates\\index.html")
	tmpl.ExecuteTemplate(w, "base.html", nil)
}

func operations(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()
		var plus = r.FormValue("+")
		var minus = r.FormValue("-")
		var multiply = r.FormValue("*")
		var divide = r.FormValue("/")
		durations := []string{plus, minus, multiply, divide}
		for i, opType := range []string{"+", "-", "*", "/"} {
			stmt, _ := db.Prepare("UPDATE Operations SET duration = ? WHERE operation = ?")
			durationInt, _ := strconv.Atoi(durations[i])
			stmt.Exec(durationInt, opType)
			stmt.Close()
		}
	}
	rows, _ := db.Query("SELECT * FROM Operations")
	columns, _ := rows.Columns()
	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtr := make([]interface{}, len(columns))
		for i := range columns {
			valuePtr[i] = &values[i]
		}
		rows.Scan(valuePtr...)
		row := make(map[string]interface{})
		for i, column := range columns {
			val := values[i]
			if val != nil {
				row[column] = val
			} else {
				row[column] = nil
			}
		}
		result = append(result, row)
	}
	data := struct {
		Title      string
		Operations []map[string]interface{}
	}{
		Title:      "Операции",
		Operations: result,
	}
	tmpl, _ := template.ParseFiles("templates\\base.html", "templates\\operations.html")
	tmpl.ExecuteTemplate(w, "base.html", data)
}

func tasks(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		stmt, _ := db.Prepare("DELETE FROM Tasks")
		stmt.Exec()
		stmt.Close()
	}
	rows, _ := db.Query("SELECT * FROM Tasks")
	columns, _ := rows.Columns()
	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtr := make([]interface{}, len(columns))
		for i := range columns {
			valuePtr[i] = &values[i]
		}
		rows.Scan(valuePtr...)
		row := make(map[string]interface{})
		for i, column := range columns {
			val := values[i]
			if val != nil {
				row[column] = val
			} else {
				row[column] = nil
			}
		}
		if row["status"] == "Успешно" {
			row["color"] = "table-success"
		} else if row["status"] == "Проводится подсчёт выражения" {
			row["color"] = "table-warning"
		} else {
			row["color"] = "table-danger"
		}
		result = append(result, row)
	}
	data := struct {
		Title string
		Tasks []map[string]interface{}
	}{
		Title: "Операции",
		Tasks: result,
	}
	tmpl, _ := template.ParseFiles("templates\\base.html", "templates\\tasks.html")
	tmpl.ExecuteTemplate(w, "base.html", data)
}

func calc(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		db.Exec("INSERT OR IGNORE INTO Calc (calc) VALUES ('')")
	}
	rows, _ := db.Query("SELECT * FROM Calc")
	columns, _ := rows.Columns()
	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtr := make([]interface{}, len(columns))
		for i := range columns {
			valuePtr[i] = &values[i]
		}
		rows.Scan(valuePtr...)
		row := make(map[string]interface{})
		for i, column := range columns {
			val := values[i]
			if val != nil {
				row[column] = val
			} else {
				row[column] = nil
			}
		}
		result = append(result, row)
	}
	data := struct {
		Title string
		Calc  []map[string]interface{}
	}{
		Title: "Операции",
		Calc:  result,
	}
	tmpl, _ := template.ParseFiles("templates\\base.html", "templates\\calc.html")
	tmpl.ExecuteTemplate(w, "base.html", data)
}

func recover_equations() {
	rows, _ := db.Query("SELECT * FROM Tasks")
	columns, _ := rows.Columns()
	ar_of_rows := make([][]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtr := make([]interface{}, len(columns))
		for i := range columns {
			valuePtr[i] = &values[i]
		}
		rows.Scan(valuePtr...)
		ar_of_rows = append(ar_of_rows, values)
	}
	rows.Close()
	for _, value := range ar_of_rows {
		if value[2] == "Проводится подсчёт выражения" {
			result := Parse_Task(delete_useless_brackets([]rune(value[1].(string))))
			if math.IsInf(result, 0) {
				stmt, _ := db.Prepare("UPDATE Tаsks SET status=?, result = ?, finish = ? WHERE task = ?")
				defer stmt.Close()
				stmt.Exec("Деление на 0", 0, time.Now().Format("01-02-2006 15:04:05"), value[1].(string))
			} else {
				stmt, _ := db.Prepare("UPDATE Tasks SET status=?, result = ?, finish = ? WHERE task = ?")
				defer stmt.Close()
				stmt.Exec("Успешно", result, time.Now().Format("01-02-2006 15:04:05"), value[1].(string))
			}
		}
	}
}

var db *sql.DB

func main() {
	if _, err := os.Stat("./data.db"); err == nil {
		db, _ = sql.Open("sqlite3", "./data.db")
		go recover_equations()
	} else if errors.Is(err, os.ErrNotExist) {
		file, _ := os.Create("data.db")
		file.Close()
		db, _ = sql.Open("sqlite3", "./data.db")
		createTable(db)
	}
	defer db.Close()
	http.HandleFunc("/operations", operations)
	http.HandleFunc("/add_task_page", add_task_page)
	http.HandleFunc("/calc", calc)
	http.HandleFunc("/tasks", tasks)
	http.ListenAndServe(":8080", nil)
}
