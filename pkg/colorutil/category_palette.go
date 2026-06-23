package colorutil

import "strings"

// CategoryColorPalette — фиксированный набор свотчей для категорий
// планируемых расходов. Должен совпадать с CATEGORY_COLORS на фронтенде
// (src/lib/palette.ts).
var CategoryColorPalette = []string{
	"#112250",
	"#3C5070",
	"#6B4226",
	"#8A6D3B",
	"#4A5D52",
	"#7D6B91",
	"#B85C5C",
	"#E0C68F",
	"#D9CBC2",
	"#9FAFA0",
}

// IsValidCategoryColor проверяет, что цвет входит в фиксированную палитру
// свотчей (сравнение без учёта регистра).
func IsValidCategoryColor(hex string) bool {
	for _, c := range CategoryColorPalette {
		if strings.EqualFold(c, hex) {
			return true
		}
	}
	return false
}
