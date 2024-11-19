package models

type TestResult struct {
	TestNumber      int      `json:"test_number"`
	Passed          bool     `json:"passed"`
	Output          string   `json:"output"`
	Input           string   `json:"input"`
	ExpectedOutput  string   `json:"expectedOutput"`
	Comments        string   `json:"comments"`
	Errors          []ErrorLine `json:"errors"`
}
