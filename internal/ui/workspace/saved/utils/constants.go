package utils

// Константы для менеджера сетки
const (
	// DefaultMinHeight - минимальная высота карточки в пикселях
	DefaultMinHeight float32 = 70

	// AnomalousHeightThreshold - порог для определения аномально большой высоты карточки
	AnomalousHeightThreshold float32 = 600

	// DefaultAnomalousHeight - разумная высота для карточек с аномально большим содержимым
	DefaultAnomalousHeight float32 = 400

	// GapSize - размер промежутка между карточками в пикселях
	GapSize float32 = 5

	// FixedCardWidth - фиксированная ширина карточки в пикселях
	FixedCardWidth float32 = 300

	// DefaultColumnCount - количество колонок по умолчанию
	DefaultColumnCount = 3

	// ScrollThreshold - порог изменения скролла для обновления макета (в пикселях)
	ScrollThreshold = 50.0

	// DebounceDelay - задержка дебаунсинга для обновления макета (в миллисекундах)
	DebounceDelay = 250

	// ThrottleInterval - интервал троттлинга для обновления макета (~30 FPS, в миллисекундах)
	ThrottleInterval = 33
)
