package ai

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	reNumUnit    = regexp.MustCompile(`^(\d+(?:[.,]\d+)?)([a-zа-яёА-ЯЁA-Z]+)$`)
	reNum        = regexp.MustCompile(`^\d+(?:[.,]\d+)?$`)
	rePlusQty    = regexp.MustCompile(`^(\d+)\+(\d+)$`)
	reDecimalComma = regexp.MustCompile(`(\d),(\d)`) // "0,5" → "0.5"
)

// unitMap maps known unit strings (lower-cased) to canonical unit values.
var unitMap = map[string]string{
	// Russian
	"кг": "kg", "кило": "kg",
	"г": "g", "гр": "g",
	"л":  "l", "литр": "l", "литра": "l", "литров": "l", "литре": "l",
	"мл": "ml",
	"пачка": "pack", "пачку": "pack", "пачки": "pack", "пачек": "pack",
	"шт": "pcs", "штук": "pcs", "штуки": "pcs", "штука": "pcs", "шт.": "pcs",
	// Latin
	"kg":    "kg",
	"g":     "g",
	"l":     "l",
	"ml":    "ml",
	"pack":  "pack", "packs": "pack", "pkg": "pack",
	"pcs":   "pcs", "pc": "pcs",
}

// ParseShoppingTextFallback parses freeform shopping list text without AI.
// Items are separated by commas, semicolons, or newlines. For each item it
// extracts an optional quantity, an optional unit, and the item name.
// X+Y promo notation (e.g. "кефир 4+2") is split into two separate items.
func ParseShoppingTextFallback(text string) []ParsedShoppingItem {
	// Normalize decimal commas ("0,5" → "0.5") before splitting on commas.
	text = reDecimalComma.ReplaceAllString(text, "$1.$2")

	var result []ParsedShoppingItem
	for _, segment := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	}) {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		result = append(result, parseOneFallbackItem(segment)...)
	}
	return result
}

func parseOneFallbackItem(raw string) []ParsedShoppingItem {
	tokens := strings.Fields(raw)
	if len(tokens) == 0 {
		return nil
	}

	// X+Y promo notation: treat as two separate items with the same name.
	for _, tok := range tokens {
		if m := rePlusQty.FindStringSubmatch(tok); m != nil {
			x, _ := strconv.ParseFloat(m[1], 64)
			y, _ := strconv.ParseFloat(m[2], 64)
			var nameParts []string
			for _, t := range tokens {
				if t != tok {
					nameParts = append(nameParts, t)
				}
			}
			name := strings.Join(nameParts, " ")
			if name == "" {
				name = raw
			}
			return []ParsedShoppingItem{
				{Name: name, Quantity: x, Unit: "pcs"},
				{Name: name, Quantity: y, Unit: "pcs"},
			}
		}
	}

	var qty float64 = -1
	var unit string
	var nameParts []string

	for _, tok := range tokens {
		tl := strings.ToLower(tok)

		// Number+unit glued together (e.g. "2кг", "500мл", "1.5л")
		if qty < 0 {
			if m := reNumUnit.FindStringSubmatch(tl); m != nil {
				if n, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64); err == nil {
					qty = n
					if canonical, ok := unitMap[m[2]]; ok {
						unit = canonical
					}
					continue
				}
			}
		}

		// Standalone number
		if qty < 0 && reNum.MatchString(tl) {
			if n, err := strconv.ParseFloat(strings.ReplaceAll(tl, ",", "."), 64); err == nil {
				qty = n
				continue
			}
		}

		// Standalone unit word (consumed at most once)
		if unit == "" {
			if canonical, ok := unitMap[tl]; ok {
				unit = canonical
				continue
			}
		}

		nameParts = append(nameParts, tok)
	}

	if qty < 0 {
		qty = 1
	}
	if unit == "" {
		unit = "pcs"
	}
	name := strings.Join(nameParts, " ")
	if name == "" {
		name = raw
	}

	return []ParsedShoppingItem{{Name: name, Quantity: qty, Unit: unit}}
}
