package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

const hmacSampleSecret = "ReAlLYSecRetKE y"

func get_current_user_id(r *http.Request) float64 {
	cookie, _ := r.Cookie("auth")
	tokenString := cookie.Value
	tokenFromString, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		}
		return []byte(hmacSampleSecret), nil
	})
	claim, _ := tokenFromString.Claims.(jwt.MapClaims)
	return claim["user_id"].(float64)
}

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
	db.Exec("CREATE TABLE IF NOT EXISTS Tasks (id INTEGER PRIMARY KEY AUTOINCREMENT, task TEXT, status TEXT, result REAL, start TEXT, finish TEXT, user_id INTEGER)")
	db.Exec("CREATE TABLE IF NOT EXISTS Operations (operation TEXT, duration INTEGER, user_id INTEGER)")
	db.Exec("CREATE TABLE IF NOT EXISTS Calc (id INTEGER PRIMARY KEY AUTOINCREMENT, calc TEXT, user_id INTEGER)")
	db.Exec("CREATE TABLE IF NOT EXISTS User (id INTEGER PRIMARY KEY AUTOINCREMENT, login TEXT, password TEXT)")
}

func check_auth(r *http.Request) bool {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return false
	}
	tokenString := cookie.Value
	tokenFromString, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("err")
		}
		return []byte(hmacSampleSecret), nil
	})
	if err != nil {
		return false
	}
	if _, ok := tokenFromString.Claims.(jwt.MapClaims); ok {
		return true
	} else {
		return false
	}
}

func add_task(task string, id int) {
	equation := strings.ReplaceAll(task, " ", "")
	equation = strings.ReplaceAll(equation, ",", ".")

	rows, _ := db.Query("SELECT * FROM Calc WHERE calc='' AND user_id=?", id)
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
		stmt, _ := db.Prepare("INSERT OR IGNORE INTO Tasks (task, status, user_id) VALUES (?, ?, ?)")
		stmt.Exec(equation, "Ожидает свободного сервера для начала вычислений", id)
		stmt.Close()
		for len(ar_of_rows) == 0 {
			time.Sleep(100 * time.Millisecond)
			rows, _ := db.Query("SELECT * FROM Calc WHERE calc='' AND user_id=?", id)
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
		stmt, _ := db.Prepare("INSERT OR IGNORE INTO Tasks (task, user_id) VALUES (?, ?)")
		stmt.Exec(equation, id)
		stmt.Close()
	}
	stmt, _ := db.Prepare("UPDATE Calc SET calc=? WHERE id=? AND user_id=?")
	stmt.Exec(equation, ar_of_rows[0][0], id)
	stmt.Close()
	if ValidEquation(equation, 0, len(equation)) && CheckBrackets(equation) {
		stmt, _ := db.Prepare("UPDATE Tasks SET status=?, start=? WHERE task = ? AND user_id=?")
		defer stmt.Close()
		stmt.Exec("Проводится подсчёт выражения", time.Now().Format("01-02-2006 15:04:05"), equation, id)
		result := Parse_Task(delete_useless_brackets([]rune(equation)))
		if math.IsInf(result, 0) {
			stmt, _ = db.Prepare("UPDATE Tasks SET status=?, result = ?, finish = ? WHERE task = ? AND user_id=?")
			stmt.Exec("Деление на 0", 0, time.Now().Format("01-02-2006 15:04:05"), equation, id)
		} else {
			stmt, _ = db.Prepare("UPDATE Tasks SET status=?, result = ?, finish = ? WHERE task = ? AND user_id=?")
			stmt.Exec("Успешно", result, time.Now().Format("01-02-2006 15:04:05"), equation, id)
		}
	} else {
		stmt, _ := db.Prepare("INSERT OR IGNORE INTO Tasks (task, status, result, start, finish, user_id) VALUES (?, ?, ?, ?, ?, ?)")
		defer stmt.Close()
		stmt.Exec(equation, "Неверный формат", 0, time.Now().Format("01-02-2006 15:04:05"), time.Now().Format("01-02-2006 15:04:05"), id)
	}
	stmt, _ = db.Prepare("UPDATE Calc SET calc=? WHERE id=? AND user_id=?")
	stmt.Exec("", ar_of_rows[0][0], id)
	stmt.Close()
}

func add_task_page(w http.ResponseWriter, r *http.Request) {
	if !check_auth(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method == "POST" {
		user_id := get_current_user_id(r)
		r.ParseForm()
		var task = r.FormValue("task")
		go add_task(task, int(user_id))
	}
	tmpl, _ := template.ParseFiles("templates\\base.html", "templates\\index.html")
	tmpl.ExecuteTemplate(w, "base.html", nil)
}

func operations(w http.ResponseWriter, r *http.Request) {
	if !check_auth(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	user_id := get_current_user_id(r)
	if r.Method == "POST" {
		r.ParseForm()
		var plus = r.FormValue("+")
		var minus = r.FormValue("-")
		var multiply = r.FormValue("*")
		var divide = r.FormValue("/")
		durations := []string{plus, minus, multiply, divide}
		for i, opType := range []string{"+", "-", "*", "/"} {
			stmt, _ := db.Prepare("UPDATE Operations SET duration = ? WHERE operation = ? AND user_id=?")
			durationInt, _ := strconv.Atoi(durations[i])
			stmt.Exec(durationInt, opType, int(user_id))
			stmt.Close()
		}
	}
	rows, _ := db.Query("SELECT * FROM Operations WHERE user_id=?", int(user_id))
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
	if !check_auth(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	user_id := get_current_user_id(r)
	if r.Method == "POST" {
		stmt, _ := db.Prepare("DELETE FROM Tasks")
		stmt.Exec()
		stmt.Close()
	}
	rows, _ := db.Query("SELECT * FROM Tasks WHERE user_id=?", int(user_id))
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
		} else if row["status"] == "Проводится подсчёт выражения" || row["status"] == "Ожидает свободного сервера для начала вычислений" {
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
	if !check_auth(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	user_id := get_current_user_id(r)
	if r.Method == "POST" {
		db.Exec("INSERT OR IGNORE INTO Calc (calc, user_id) VALUES ('', ?)", int(user_id))
	}
	rows, _ := db.Query("SELECT * FROM Calc WHERE user_id=?", int(user_id))
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
		if value[2] == "Ожидает свободного сервера для начала вычислений" {
			rows1, _ := db.Query("SELECT * FROM Calc WHERE calc=''")
			columns1, _ := rows1.Columns()
			ar_of_rows1 := make([][]interface{}, 0)
			for rows1.Next() {
				values1 := make([]interface{}, len(columns1))
				valuePtr1 := make([]interface{}, len(columns1))
				for i := range columns1 {
					valuePtr1[i] = &values1[i]
				}
				rows1.Scan(valuePtr1...)
				ar_of_rows1 = append(ar_of_rows1, values1)
			}
			rows1.Close()
			for len(ar_of_rows1) == 0 {
				time.Sleep(100 * time.Millisecond)
				rows1, _ := db.Query("SELECT * FROM Calc WHERE calc=''")
				columns1, _ := rows1.Columns()
				ar_of_rows1 = make([][]interface{}, 0)
				for rows1.Next() {
					values1 := make([]interface{}, len(columns1))
					valuePtr1 := make([]interface{}, len(columns1))
					for i := range columns1 {
						valuePtr1[i] = &values1[i]
					}
					rows1.Scan(valuePtr1...)
					ar_of_rows1 = append(ar_of_rows1, values1)
				}
			}
			stmt, _ := db.Prepare("UPDATE Calc SET calc=? WHERE id=?")
			stmt.Exec(value[1], ar_of_rows[0][0])
			stmt.Close()
			stmt, _ = db.Prepare("UPDATE Tasks SET status=?, start=? WHERE task = ?")
			stmt.Exec("Проводится подсчёт выражения", time.Now().Format("01-02-2006 15:04:05"), value[1])
			stmt.Close()
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
			stmt, _ = db.Prepare("UPDATE Calc SET calc=? WHERE calc=?")
			stmt.Exec("", value[1])
			stmt.Close()
		} else if value[2] == "Проводится подсчёт выражения" {
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
			stmt, _ := db.Prepare("UPDATE Calc SET calc=? WHERE calc=?")
			stmt.Exec("", value[1])
			stmt.Close()
		}
	}
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()
		var login = r.FormValue("login")
		var password = r.FormValue("password")
		type User struct {
			id       *int
			login    *string
			password *string
		}
		user := User{}
		row := db.QueryRow("SELECT * FROM User WHERE login=?", login)
		row.Scan(&user.id, &user.login, &user.password)
		if user.id == nil {
			http.Redirect(w, r, "/register", http.StatusSeeOther)
			return
		}
		if CheckPasswordHash(password, *user.password) {
			now := time.Now()
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"user_id": user.id,
				"nbf":     now.Unix(),
				"exp":     now.Add(60 * time.Minute).Unix(),
				"iat":     now.Unix(),
			})
			tokenString, _ := token.SignedString([]byte(hmacSampleSecret))

			cookie := http.Cookie{
				Name:     "auth",
				Value:    tokenString,
				MaxAge:   0,
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			}
			http.SetCookie(w, &cookie)

			http.Redirect(w, r, "/add_task_page", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
	}
	tmpl, _ := template.ParseFiles("templates\\base.html", "templates\\login.html")
	tmpl.ExecuteTemplate(w, "base.html", nil)
}

func register(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()
		var login = r.FormValue("login")
		var password = r.FormValue("password")
		stmt, _ := db.Prepare("INSERT OR IGNORE INTO User (login, password) VALUES (?, ?)")
		defer stmt.Close()
		password_hashed, _ := HashPassword(password)
		type User struct {
			id       *int
			login    *string
			password *string
		}
		user := User{}
		row := db.QueryRow("SELECT * FROM User WHERE login=?", login)
		row.Scan(&user.id, &user.login, &user.password)
		if user.id != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		result, _ := stmt.Exec(login, password_hashed)
		new_user_id, _ := result.LastInsertId()
		db.Exec("INSERT OR IGNORE INTO Operations (operation, duration, user_id) VALUES ('+', 1, ?)", new_user_id)
		db.Exec("INSERT OR IGNORE INTO Operations (operation, duration, user_id) VALUES ('-', 1, ?)", new_user_id)
		db.Exec("INSERT OR IGNORE INTO Operations (operation, duration, user_id) VALUES ('*', 1, ?)", new_user_id)
		db.Exec("INSERT OR IGNORE INTO Operations (operation, duration, user_id) VALUES ('/', 1, ?)", new_user_id)
		now := time.Now()
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": new_user_id,
			"nbf":     now.Unix(),
			"exp":     now.Add(60 * time.Minute).Unix(),
			"iat":     now.Unix(),
		})
		tokenString, _ := token.SignedString([]byte(hmacSampleSecret))
		cookie := http.Cookie{
			Name:     "auth",
			Value:    tokenString,
			MaxAge:   0,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, &cookie)

		http.Redirect(w, r, "/add_task_page", http.StatusSeeOther)
	}
	tmpl, _ := template.ParseFiles("templates\\base.html", "templates\\register.html")
	tmpl.ExecuteTemplate(w, "base.html", nil)
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
	http.Handle("/operations", http.HandlerFunc(operations))
	http.Handle("/add_task_page", http.HandlerFunc(add_task_page))
	http.Handle("/calc", http.HandlerFunc(calc))
	http.Handle("/tasks", http.HandlerFunc(tasks))
	http.HandleFunc("/login", login)
	http.HandleFunc("/register", register)
	http.ListenAndServe(":8080", nil)
}
