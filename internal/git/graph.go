package git

import (
	"encoding/json"
	"fmt"

	"github.com/alaingilbert/git2graph/git2graph"
)

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

// toInt безопасно конвертирует interface{} в int.
// Поддерживает float64 (стандартный тип JSON), int и json.Number.
func toInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case json.Number:
		n, _ := val.Int64()
		return int(n)
	default:
		return 0
	}
}

// BuildGraphRows принимает коммиты с hash и parents, вычисляет данные графа
// через git2graph и возвращает коммиты с заполненным полем GraphData.
func BuildGraphRows(commits []Commit) ([]Commit, error) {
	if len(commits) == 0 {
		return commits, nil
	}

	// Собираем set всех хешей в наборе — parent'ы за пределами набора
	// вызывают панику в git2graph (index out of range)
	knownHashes := make(map[string]struct{}, len(commits))
	for _, c := range commits {
		knownHashes[c.Hash] = struct{}{}
	}

	// Конвертируем коммиты в формат ввода git2graph,
	// фильтруя parent'ов которых нет в наборе (при пагинации)
	input := make([]map[string]interface{}, len(commits))
	for i, c := range commits {
		parents := make([]interface{}, 0, len(c.Parents))
		for _, p := range c.Parents {
			if _, ok := knownHashes[p]; ok {
				parents = append(parents, p)
			}
		}
		input[i] = map[string]interface{}{
			"id":      c.Hash,
			"parents": parents,
		}
	}

	// Сериализуем в JSON для git2graph
	jsonData, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("ошибка маршалинга входных данных: %w", err)
	}

	nodes, err := git2graph.GetInputNodesFromJSON(jsonData)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга узлов git2graph: %w", err)
	}

	out, err := git2graph.GetRows(nodes)
	if err != nil {
		return nil, fmt.Errorf("ошибка вычисления графа: %w", err)
	}

	// Парсим результат и заполняем GraphData для каждого коммита
	for idx, node := range out.Nodes {
		if idx >= len(commits) {
			break
		}

		gRaw, ok := (*node)["g"]
		if !ok {
			continue
		}

		// "g" — это []any{x, color, lines}
		// Но данные хранятся как Go-типы, поэтому сериализуем/десериализуем через JSON
		gJSON, err := json.Marshal(gRaw)
		if err != nil {
			continue
		}

		var gArr []json.RawMessage
		if err := json.Unmarshal(gJSON, &gArr); err != nil {
			continue
		}
		if len(gArr) < 3 {
			continue
		}

		// Парсим column (x)
		var column float64
		if err := json.Unmarshal(gArr[0], &column); err != nil {
			continue
		}

		// Парсим color
		var nodeColor string
		if err := json.Unmarshal(gArr[1], &nodeColor); err != nil {
			continue
		}

		// Парсим lines — массив массивов [x1, x2, type, color]
		var rawLines []json.RawMessage
		if err := json.Unmarshal(gArr[2], &rawLines); err != nil {
			continue
		}

		lines := make([]GraphLine, 0, len(rawLines))
		for _, rl := range rawLines {
			var lineArr []json.RawMessage
			if err := json.Unmarshal(rl, &lineArr); err != nil || len(lineArr) < 4 {
				continue
			}

			var x1, x2 float64
			var lineType float64
			var lineColor string

			json.Unmarshal(lineArr[0], &x1)
			json.Unmarshal(lineArr[1], &x2)
			json.Unmarshal(lineArr[2], &lineType)
			json.Unmarshal(lineArr[3], &lineColor)

			lines = append(lines, GraphLine{
				X1:    int(x1),
				X2:    int(x2),
				Type:  int(lineType),
				Color: lineColor,
			})
		}

		commits[idx].GraphData = &GraphData{
			Column: int(column),
			Color:  nodeColor,
			Lines:  lines,
		}
	}

	return commits, nil
}
