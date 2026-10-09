package personal

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var amountPat = regexp.MustCompile(`^[-+]?\d+(?:\.\d{1,2})?$`)

type SpendRow struct {
	ID          string  `json:"id"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
}

type SpendReport struct {
	Income           float64         `json:"income"`
	Spending         float64         `json:"spending"`
	Saved            float64         `json:"saved"`
	Count            int             `json:"count"`
	Categories       []SpendCategory `json:"categories"`
	Transactions     []SpendRow      `json:"transactions"`
	Period           SpendPeriod     `json:"period"`
	AmountConvention string          `json:"amount_convention"`
}

type SpendCategory struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

type SpendPeriod struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func analyzeSpending(csv string) (SpendReport, error) {
	var zero SpendReport
	if len(csv) > 500000 {
		return zero, fmt.Errorf("personal: import at most 500 KB of transaction CSV")
	}
	if strings.HasPrefix(csv, "\ufeff") {
		csv = strings.TrimPrefix(csv, "\ufeff")
	}
	rows, err := parseCSV(csv)
	if err != nil {
		return zero, err
	}
	if len(rows) == 0 {
		return zero, fmt.Errorf("personal: CSV needs date,description,amount,category columns")
	}
	header := make([]string, len(rows[0]))
	for i, v := range rows[0] {
		header[i] = strings.ToLower(strings.TrimSpace(v))
	}
	need := []string{"date", "description", "amount", "category"}
	idx := map[string]int{}
	for i, h := range header {
		if _, ok := idx[h]; ok {
			for _, n := range need {
				if h == n {
					return zero, fmt.Errorf("personal: CSV has a duplicate required column")
				}
			}
		}
		idx[h] = i
	}
	for _, n := range need {
		if _, ok := idx[n]; !ok {
			return zero, fmt.Errorf("personal: CSV needs date,description,amount,category columns")
		}
	}
	body := rows[1:]
	if len(body) == 0 || len(body) > 5000 {
		return zero, fmt.Errorf("personal: import between 1 and 5,000 transactions")
	}
	tx := make([]SpendRow, 0, len(body))
	var expenses, income int64
	grouped := map[string]int64{}
	dates := make([]string, 0, len(body))
	for i, r := range body {
		if len(r) != len(header) {
			return zero, fmt.Errorf("personal: row %d has the wrong number of columns", i+2)
		}
		get := func(name string) string { return strings.TrimSpace(r[idx[name]]) }
		date, amount := get("date"), get("amount")
		if _, err := time.Parse("2006-01-02", date); err != nil || !amountPat.MatchString(amount) {
			return zero, fmt.Errorf("personal: row %d needs an ISO date and a plain amount with at most two decimal places", i+2)
		}
		n, _ := strconv.ParseFloat(amount, 64)
		cents := int64(math.Round(n * 100))
		if cents > 1e12 || cents < -1e12 {
			return zero, fmt.Errorf("personal: transaction amount is out of range")
		}
		cat := get("category")
		if cat == "" {
			cat = "Uncategorized"
		}
		tx = append(tx, SpendRow{
			ID:          fmt.Sprintf("row-%d", i+2),
			Date:        date,
			Description: get("description"),
			Amount:      float64(cents) / 100,
			Category:    cat,
		})
		if cents > 0 {
			expenses += cents
			grouped[cat] += cents
		} else if cents < 0 {
			income += -cents
		}
		dates = append(dates, date)
	}
	cats := make([]SpendCategory, 0, len(grouped))
	for name, cents := range grouped {
		cats = append(cats, SpendCategory{Name: name, Amount: float64(cents) / 100})
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i].Amount > cats[j].Amount })
	sort.Strings(dates)
	return SpendReport{
		Income:           float64(income) / 100,
		Spending:         float64(expenses) / 100,
		Saved:            float64(income-expenses) / 100,
		Count:            len(tx),
		Categories:       cats,
		Transactions:     tx,
		Period:           SpendPeriod{From: dates[0], To: dates[len(dates)-1]},
		AmountConvention: "Positive expenses; negative income. Values use the source currency; no currency conversion.",
	}, nil
}

func parseCSV(s string) ([][]string, error) {
	var rows [][]string
	var row []string
	var cell strings.Builder
	quoted := false
	for i := 0; i <= len(s); i++ {
		var c byte
		if i < len(s) {
			c = s[i]
		}
		if c == '"' {
			if quoted && i+1 < len(s) && s[i+1] == '"' {
				cell.WriteByte('"')
				i++
				continue
			}
			if !quoted && cell.Len() > 0 {
				return nil, fmt.Errorf("personal: invalid quoted CSV field")
			}
			quoted = !quoted
			if !quoted {
				if i+1 < len(s) {
					next := s[i+1]
					if next != ',' && next != '\n' && !(next == '\r' && (i+2 >= len(s) || s[i+2] == '\n')) {
						return nil, fmt.Errorf("personal: invalid quoted CSV field")
					}
				}
			}
			continue
		}
		if !quoted && (c == ',' || c == '\n' || i == len(s)) {
			val := strings.TrimSuffix(cell.String(), "\r")
			row = append(row, val)
			cell.Reset()
			if c != ',' {
				keep := false
				for _, v := range row {
					if strings.TrimSpace(v) != "" {
						keep = true
						break
					}
				}
				if keep {
					rows = append(rows, row)
				}
				row = nil
			}
			continue
		}
		if i < len(s) {
			cell.WriteByte(c)
		}
	}
	if quoted {
		return nil, fmt.Errorf("personal: CSV has an unclosed quoted field")
	}
	return rows, nil
}
