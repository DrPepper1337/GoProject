package agent

import (
	"database/sql"
	"strconv"
	"sync"
	"time"
)

func OperationTime(operation string, db *sql.DB, user_id int) int {
	rows, _ := db.Query("SELECT duration FROM Operations WHERE operation = ? AND user_id=?", operation, user_id)
	defer rows.Close()

	var duration int
	if rows.Next() {
		rows.Scan(&duration)
	}
	return duration
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

func Delete_useless_brackets(end_str []rune) []rune {
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

func Parse_Task(task []rune, db *sql.DB, user_id int) float64 {
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
					n1 = Parse_Task(Delete_useless_brackets([]rune(current_string)), db, user_id)
				}()
				go func() {
					defer wg.Done()
					n2 = Parse_Task(Delete_useless_brackets(end_str), db, user_id)
				}()
				wg.Wait()
				time.Sleep(time.Duration(OperationTime("+", db, user_id)) * time.Millisecond)
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
					n1 = Parse_Task(Delete_useless_brackets([]rune(current_string)), db, user_id)
				}()
				go func() {
					defer wg.Done()
					n2 = Parse_Task(Delete_useless_brackets(end_str), db, user_id)
				}()
				wg.Wait()
				time.Sleep(time.Duration(OperationTime("-", db, user_id)) * time.Millisecond)
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
						n1 = Parse_Task(Delete_useless_brackets([]rune(current_string)), db, user_id)
					}()
					go func() {
						defer wg.Done()
						n2 = Parse_Task(Delete_useless_brackets(end_str), db, user_id)
					}()
					wg.Wait()
					time.Sleep(time.Duration(OperationTime("*", db, user_id)) * time.Millisecond)
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
						n1 = Parse_Task(Delete_useless_brackets([]rune(current_string)), db, user_id)
					}()
					go func() {
						defer wg.Done()
						n2 = Parse_Task(Delete_useless_brackets(end_str), db, user_id)
					}()
					wg.Wait()
					time.Sleep(time.Duration(OperationTime("/", db, user_id)) * time.Millisecond)
					return division(n1, n2)
				}
				current_string += string(letter)
			}
		}
	}
	return 1.0
}
