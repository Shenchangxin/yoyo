package personal

import "testing"

func TestAnalyzeSpending(t *testing.T) {
	csv := "date,description,amount,category\n2026-01-01,Coffee,4.50,Food\n2026-01-02,Paycheck,-2000.00,Income\n2026-01-03,Rent,900,Housing\n"
	r, err := analyzeSpending(csv)
	if err != nil {
		t.Fatal(err)
	}
	if r.Count != 3 || r.Spending != 904.5 || r.Income != 2000 || r.Saved != 1095.5 {
		t.Fatalf("%+v", r)
	}
	if r.Period.From != "2026-01-01" || r.Period.To != "2026-01-03" {
		t.Fatalf("%+v", r.Period)
	}
	if len(r.Categories) != 2 || r.Categories[0].Name != "Housing" {
		t.Fatalf("%+v", r.Categories)
	}
}

func TestAnalyzeSpendingBOMAndQuotes(t *testing.T) {
	csv := "\ufeffdate,description,amount,category\n2026-02-01,\"Bag, large\",12.00,Food\n"
	r, err := analyzeSpending(csv)
	if err != nil || r.Count != 1 || r.Transactions[0].Description != "Bag, large" {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestAnalyzeSpendingRejects(t *testing.T) {
	if _, err := analyzeSpending("nope"); err == nil {
		t.Fatal("header")
	}
	if _, err := analyzeSpending("date,description,amount,category\n"); err == nil {
		t.Fatal("empty")
	}
	if _, err := analyzeSpending("date,description,amount,category\n01/02/2026,x,1,y\n"); err == nil {
		t.Fatal("date")
	}
}
