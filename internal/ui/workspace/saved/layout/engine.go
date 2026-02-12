package layout

import (
	"fmt"
	"time"
	
	"projectT/internal/ui/workspace/saved/models"
)

// LayoutEngine реализует алгоритм размещения для расчёта позиций карточек в сетке
type LayoutEngine struct {
	// Текущая высота колонок - используем срез вместо фиксированного массива
	columnHeights []float32
}

// NewLayoutEngine создает новый движок раскладки
func NewLayoutEngine() *LayoutEngine {
	// Инициализируем высоты 3 колонок нулями по умолчанию
	defaultColumnCount := 3
	columnHeights := make([]float32, defaultColumnCount)
	for i := 0; i < defaultColumnCount; i++ {
		columnHeights[i] = 0
	}

	return &LayoutEngine{
		columnHeights: columnHeights,
	}
}

// CalculatePositions рассчитывает позиции карточек в сетке с переменным количеством колонок
func (le *LayoutEngine) CalculatePositions(cards []*models.CardInfo, availableColumns int) []models.CellPosition {
	startTime := time.Now()
	fmt.Printf("[%s] Starting position calculation for %d cards with %d available columns\n", time.Now().Format("15:04:05.000"), len(cards), availableColumns)
	
	positions := make([]models.CellPosition, len(cards))

	// Инициализируем высоты колонок в зависимости от доступного количества колонок
	// Если availableColumns <= 0, используем значение по умолчанию
	if availableColumns <= 0 {
		availableColumns = 3
	}

	// Изменяем размер среза высот колонок в соответствии с доступным количеством колонок
	initStart := time.Now()
	le.columnHeights = make([]float32, availableColumns)
	for i := 0; i < availableColumns; i++ {
		le.columnHeights[i] = 0
	}
	fmt.Printf("[%s] Column heights initialization took %v\n", time.Now().Format("15:04:05.000"), time.Since(initStart))

	cardProcessingStart := time.Now()
	for i, card := range cards {
		// Находим колонку с наименьшей высотой
		findColStart := time.Now()
		minHeight := le.columnHeights[0]
		minIndex := 0

		for j := 1; j < availableColumns; j++ {
			if le.columnHeights[j] < minHeight {
				minHeight = le.columnHeights[j]
				minIndex = j
			}
		}
		fmt.Printf("[%s] Column finding for card %d took %v\n", time.Now().Format("15:04:05.000"), i, time.Since(findColStart))

		// Устанавливаем позицию: X - номер колонки (0, 1, ..., N-1), Y - текущая высота колонки
		positions[i] = models.CellPosition{X: minIndex, Y: int(minHeight)}

		// Вычисляем фактическую высоту карточки
		cardHeight := card.ActualHeight

		// Если ActualHeight не установлен, используем минимальную высоту
		if cardHeight <= 0 {
			cardHeight = 70 // минимальная высота по умолчанию
		}

		// Обновляем высоту выбранной колонки
		le.columnHeights[minIndex] = minHeight + cardHeight + 5 // 5 - размер промежутка

		// Обновляем ActualHeight в карточке, если не установлено
		if card.ActualHeight <= 0 {
			card.ActualHeight = cardHeight
		}
	}
	fmt.Printf("[%s] Card position processing took %v\n", time.Now().Format("15:04:05.000"), time.Since(cardProcessingStart))

	fmt.Printf("[%s] Position calculation completed in %v\n", time.Now().Format("15:04:05.000"), time.Since(startTime))
	return positions
}

// UpdateColumnHeights позволяет обновить высоты колонок вручную
func (le *LayoutEngine) UpdateColumnHeights(heights []float32) {
	le.columnHeights = heights
}

// GetColumnHeights возвращает текущие высоты колонок
func (le *LayoutEngine) GetColumnHeights() []float32 {
	return le.columnHeights
}
