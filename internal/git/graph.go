package git

// GraphLine описывает одну линию в строке графа.
type GraphLine struct {
	X1    int    `json:"x1"`
	X2    int    `json:"x2"`
	Type  int    `json:"type"`
	Color string `json:"color"`
}

// GraphData содержит данные графа для одного коммита.
type GraphData struct {
	Column int         `json:"column"`
	Color  string      `json:"color"`
	Lines  []GraphLine `json:"lines"`
}
