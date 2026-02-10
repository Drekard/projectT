package cards

import "encoding/json"

// Block определяет структуру блока контента
type Block struct {
	Type         string `json:"type"`
	Content      string `json:"content,omitempty"`
	FileHash     string `json:"file_hash,omitempty"`
	OriginalName string `json:"original_name,omitempty"`
	Extension    string `json:"extension,omitempty"`
	Description  string `json:"description,omitempty"`
}

// ParseBlocks парсит JSON-строку в массив блоков
func ParseBlocks(contentMeta string) ([]Block, error) {
	var blocks []Block
	err := json.Unmarshal([]byte(contentMeta), &blocks)
	return blocks, err
}

// CountBlocksByType подсчитывает количество блоков каждого типа
func CountBlocksByType(contentMeta string) map[string]int {
	counts := make(map[string]int)
	blocks, err := ParseBlocks(contentMeta)
	if err != nil {
		return counts
	}

	for _, block := range blocks {
		counts[block.Type]++
	}

	return counts
}
