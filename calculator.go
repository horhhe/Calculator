package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type Request struct {
	Expression string `json:"expression"`
}

type Response struct {
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func main() {
	http.HandleFunc("/api/v1/calculate", calculateHandler)
	http.ListenAndServe(":8080", nil)
}

func calculateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := Calc(req.Expression)
	if err != nil {
		if errors.Is(err, errors.New("empty expression")) {
			http.Error(w, `{"error": "Expression is not valid"}`, http.StatusUnprocessableEntity)
		} else {
			http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	response := Response{Result: strconv.FormatFloat(result, 'f', -1, 64)}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func Calc(expression string) (float64, error) {
	expression = strings.ReplaceAll(expression, " ", "")
	if expression == "" {
		return 0, errors.New("empty expression")
	}

	precedence := map[rune]int{
		'+': 1,
		'-': 1,
		'*': 2,
		'/': 2,
	}
	output := []string{}
	operators := []rune{}

	var numberBuffer strings.Builder
	for _, ch := range expression {
		if (ch >= '0' && ch <= '9') || ch == '.' {
			numberBuffer.WriteRune(ch)
		} else {
			if numberBuffer.Len() > 0 {
				output = append(output, numberBuffer.String())
				numberBuffer.Reset()
			}
			if ch == '(' {
				operators = append(operators, ch)
			} else if ch == ')' {
				for len(operators) > 0 && operators[len(operators)-1] != '(' {
					output = append(output, string(operators[len(operators)-1]))
					operators = operators[:len(operators)-1]
				}
				if len(operators) == 0 {
					return 0, errors.New("mismatched parentheses")
				}
				operators = operators[:len(operators)-1]
			} else if precedence[ch] > 0 {
				for len(operators) > 0 && precedence[operators[len(operators)-1]] >= precedence[ch] {
					output = append(output, string(operators[len(operators)-1]))
					operators = operators[:len(operators)-1]
				}
				operators = append(operators, ch)
			} else {
				return 0, errors.New("invalid character: " + string(ch))
			}
		}
	}
	if numberBuffer.Len() > 0 {
		output = append(output, numberBuffer.String())
		numberBuffer.Reset()
	}
	for len(operators) > 0 {
		if operators[len(operators)-1] == '(' {
			return 0, errors.New("mismatched parentheses")
		}
		output = append(output, string(operators[len(operators)-1]))
		operators = operators[:len(operators)-1]
	}

	return evaluatePostfix(output)
}

func evaluatePostfix(tokens []string) (float64, error) {
	stack := []float64{}

	for _, token := range tokens {
		if isNumber(token) {
			num, err := strconv.ParseFloat(token, 64)
			if err != nil {
				return 0, err
			}
			stack = append(stack, num)
		} else {
			if len(stack) < 2 {
				return 0, errors.New("not enough operands for operation: " + token)
			}
			b := stack[len(stack)-1]
			a := stack[len(stack)-2]
			stack = stack[:len(stack)-2]
			switch token {
			case "+":
				stack = append(stack, a+b)
			case "-":
				stack = append(stack, a-b)
			case "*":
				stack = append(stack, a*b)
			case "/":
				if b == 0 {
					return 0, errors.New("division by zero")
				}
				stack = append(stack, a/b)
			default:
				return 0, errors.New("invalid operation: " + token)
			}
		}
	}

	if len(stack) != 1 {
		return 0, errors.New("invalid expression")
	}

	return stack[0], nil
}

func isNumber(token string) bool {
	_, err := strconv.ParseFloat(token, 64)
	return err == nil
}
