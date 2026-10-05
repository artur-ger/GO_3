package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        time.Time
}

// Budget — лимит расходов по категории.
// Period описывает период (месяц, год); проверка сравнивает сумму транзакций с фиксированным Limit.
type Budget struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
	Period   string  `json:"period,omitempty"`
}

var transactions []Transaction

// Начальные бюджеты: пара категорий до загрузки файла.
var budgets = map[string]Budget{
	"еда":       {Category: "еда", Limit: 5000, Period: "месяц"},
	"транспорт": {Category: "транспорт", Limit: 2000, Period: "месяц"},
}

// SetBudget добавляет бюджет или заменяет лимит существующей категории.
func SetBudget(b Budget) {
	if budgets == nil {
		budgets = make(map[string]Budget)
	}
	budgets[b.Category] = b
}

// AddTransaction сохраняет транзакцию.
// Если для категории задан бюджет и новая сумма его превышает, транзакция не записывается.
func AddTransaction(tx Transaction) error {
	if tx.Amount == 0 {
		return errors.New("amount must not be zero")
	}
	if strings.TrimSpace(tx.Category) == "" {
		return errors.New("category must not be empty")
	}

	if b, ok := budgets[tx.Category]; ok {
		total := categorySpent(tx.Category) + tx.Amount
		if total > b.Limit {
			return fmt.Errorf("budget exceeded: category %q, limit %.2f, total %.2f", tx.Category, b.Limit, total)
		}
	}

	if tx.Date.IsZero() {
		tx.Date = time.Now()
	}

	tx.ID = len(transactions) + 1
	transactions = append(transactions, tx)
	return nil
}

func categorySpent(category string) float64 {
	var sum float64
	for _, tx := range transactions {
		if tx.Category == category {
			sum += tx.Amount
		}
	}
	return sum
}

func ListTransactions() []Transaction {
	result := make([]Transaction, len(transactions))
	copy(result, transactions)
	return result
}

// LoadBudgets читает JSON-массив бюджетов и записывает каждый через SetBudget.
func LoadBudgets(r io.Reader) error {
	if r == nil {
		return errors.New("budget reader is nil")
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read budgets: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return errors.New("budgets data is empty")
	}

	var list []Budget
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("parse budgets json: %w", err)
	}

	for i, b := range list {
		if strings.TrimSpace(b.Category) == "" {
			return fmt.Errorf("budget %d: category must not be empty", i+1)
		}
		if b.Limit < 0 {
			return fmt.Errorf("budget %q: limit must not be negative", b.Category)
		}
		SetBudget(b)
	}
	return nil
}

func main() {
	fmt.Println("Ledger service started")

	SetBudget(Budget{Category: "еда", Limit: 5000, Period: "месяц"})
	SetBudget(Budget{Category: "транспорт", Limit: 2000, Period: "месяц"})

	file, err := os.Open("budgets.json")
	if err != nil {
		fmt.Println("Ошибка открытия budgets.json:", err)
		os.Exit(1)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	if err := LoadBudgets(reader); err != nil {
		fmt.Println("Ошибка загрузки бюджетов:", err)
		os.Exit(1)
	}

	fmt.Println("Бюджеты:")
	categories := make([]string, 0, len(budgets))
	for category := range budgets {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	for _, category := range categories {
		b := budgets[category]
		fmt.Printf("  %s: лимит %.2f (%s)\n", b.Category, b.Limit, b.Period)
	}

	testTransactions := []Transaction{
		{Amount: 1200, Category: "еда", Description: "продукты", Date: time.Date(2026, 10, 5, 0, 0, 0, 0, time.Local)},
		{Amount: 4000, Category: "еда", Description: "ужин в ресторане", Date: time.Date(2026, 10, 5, 0, 0, 0, 0, time.Local)},
		{Amount: 2000, Category: "транспорт", Description: "метро", Date: time.Date(2026, 10, 5, 0, 0, 0, 0, time.Local)},
		{Amount: 0, Category: "прочее", Description: "Некорректная транзакция"},
	}

	for _, tx := range testTransactions {
		if err := AddTransaction(tx); err != nil {
			fmt.Printf("Ошибка добавления транзакции %q: %v\n", tx.Description, err)
			continue
		}
		saved := transactions[len(transactions)-1]
		fmt.Printf("Добавлена транзакция #%d: %s %.2f\n", saved.ID, saved.Category, saved.Amount)
	}

	fmt.Println("Список транзакций:")
	for _, tx := range ListTransactions() {
		fmt.Printf("  #%d | %s | %-12s | %8.2f | %s\n",
			tx.ID, tx.Date.Format("2006-01-02"), tx.Category, tx.Amount, tx.Description)
	}
}
