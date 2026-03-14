package page

type PageProps struct {
	Params       map[string]any `json:"params"`
	SearchParams map[string]any `json:"searchParams"`
}
