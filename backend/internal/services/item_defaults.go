package services

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kincart/internal/models"
)

// ItemDefaults is the unit and category an item name should be prefilled with.
// A zero ItemDefaults means "nothing known" — the caller keeps its own defaults
// (`pcs` / uncategorized) rather than treating this as an instruction to clear.
type ItemDefaults struct {
	Unit       string
	CategoryID *uuid.UUID
}

// Known reports whether anything at all was resolved.
func (d ItemDefaults) Known() bool {
	return d.Unit != "" || d.CategoryID != nil
}

// ResolveItemDefaults resolves the unit and category to prefill for a single item
// name, from this family's purchase history.
//
// Unit is resolved per-shop: an alias recorded at shopID wins, because the same
// item is routinely a pack at one shop and loose at another. Category ignores shop
// entirely — where a thing belongs in the aisle layout does not change per store.
//
// History only. The AI fallback for never-seen items lives at the call sites that
// can afford it (paste preview, receipt processing), not here, so that a synchronous
// add can call this freely without risking a network round-trip.
func ResolveItemDefaults(ctx context.Context, tx *gorm.DB, familyID uuid.UUID, name string, shopID *uuid.UUID) (ItemDefaults, error) {
	byName, err := ResolveItemDefaultsBatch(ctx, tx, familyID, []string{name}, shopID)
	if err != nil {
		return ItemDefaults{}, err
	}
	return byName[strings.ToLower(name)], nil
}

// ResolveItemDefaultsBatch resolves defaults for many names in a single query, keyed
// by the Go-lowercased name. Callers resolving more than one item MUST use this rather
// than looping over ResolveItemDefaults, which would issue one query per name.
func ResolveItemDefaultsBatch(ctx context.Context, tx *gorm.DB, familyID uuid.UUID, names []string, shopID *uuid.UUID) (map[string]ItemDefaults, error) {
	out := make(map[string]ItemDefaults, len(names))
	if len(names) == 0 {
		return out, nil
	}

	// Go-lowercased, matched against the indexed planned_name_lower column. SQLite's
	// LOWER() folds ASCII only, so a SQL-side comparison would silently miss every
	// Cyrillic and accented Czech name — exactly the families this feature is for.
	lowered := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		l := strings.ToLower(n)
		if l == "" {
			continue
		}
		if _, dup := seen[l]; dup {
			continue
		}
		seen[l] = struct{}{}
		lowered = append(lowered, l)
	}
	if len(lowered) == 0 {
		return out, nil
	}

	var aliases []models.ItemAlias
	if err := tx.WithContext(ctx).
		Where("family_id = ? AND planned_name_lower IN ?", familyID, lowered).
		Find(&aliases).Error; err != nil {
		return nil, err
	}

	grouped := make(map[string][]models.ItemAlias, len(lowered))
	for _, a := range aliases {
		grouped[a.PlannedNameLower] = append(grouped[a.PlannedNameLower], a)
	}
	for name, group := range grouped {
		out[name] = ItemDefaultsFromAliases(group, shopID)
	}

	// Drop any resolved category whose category row no longer exists — a category
	// deleted after the alias recorded it leaves a dangling id. Prefilling it would
	// make a valid add fail validation, or save a receipt item pointing at nothing.
	if err := dropDeadCategories(ctx, tx, familyID, out); err != nil {
		return nil, err
	}
	return out, nil
}

// dropDeadCategories nils out any CategoryID in the results that does not match a
// live category for the family. One query, only when at least one category resolved.
func dropDeadCategories(ctx context.Context, tx *gorm.DB, familyID uuid.UUID, out map[string]ItemDefaults) error {
	ids := make([]uuid.UUID, 0, len(out))
	for _, d := range out {
		if d.CategoryID != nil {
			ids = append(ids, *d.CategoryID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	var liveIDs []uuid.UUID
	if err := tx.WithContext(ctx).Model(&models.Category{}).
		Where("family_id = ? AND id IN ?", familyID, ids).
		Pluck("id", &liveIDs).Error; err != nil {
		return err
	}
	live := make(map[uuid.UUID]bool, len(liveIDs))
	for _, id := range liveIDs {
		live[id] = true
	}

	for name, d := range out {
		if d.CategoryID != nil && !live[*d.CategoryID] {
			d.CategoryID = nil
			out[name] = d
		}
	}
	return nil
}

// ItemDefaultsFromAliases derives the defaults from aliases the caller has already
// loaded. ParseListText batch-loads aliases for the whole paste to build its price
// hints and variants; it applies the same rules through this rather than re-querying
// per item, which would undo that batching.
func ItemDefaultsFromAliases(group []models.ItemAlias, shopID *uuid.UUID) ItemDefaults {
	return ItemDefaults{
		Unit:       resolveUnit(group, shopID),
		CategoryID: resolveCategory(group),
	}
}

// LoadFamilyCategories returns the family's categories in display order. Callers
// pass the result to both the AI (as the set of names it may choose from) and
// MatchCategoryName (to resolve what it chose), so a single load serves both.
func LoadFamilyCategories(ctx context.Context, tx *gorm.DB, familyID uuid.UUID) ([]models.Category, error) {
	var categories []models.Category
	if err := tx.WithContext(ctx).
		Where("family_id = ?", familyID).
		Order("sort_order asc").
		Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

// CategoryNames extracts the names to offer the AI as its allowed choices.
func CategoryNames(categories []models.Category) []string {
	names := make([]string, 0, len(categories))
	for _, c := range categories {
		if c.Name != "" {
			names = append(names, c.Name)
		}
	}
	return names
}

// MatchCategoryName resolves a category name to one of the family's category rows,
// returning nil when nothing matches. An unmatched name leaves the item
// uncategorized; this never creates a category.
//
// Comparison is Go-side lowercased, never SQL LOWER(): SQLite's LOWER() folds ASCII
// only, so "Молочное" and "молочное" would not compare equal and the match would
// silently fail for exactly the non-English families the constrained-enum design
// exists to serve.
func MatchCategoryName(categories []models.Category, name string) *uuid.UUID {
	want := strings.ToLower(strings.TrimSpace(name))
	if want == "" {
		return nil
	}
	for i := range categories {
		if strings.ToLower(strings.TrimSpace(categories[i].Name)) == want {
			id := categories[i].ID
			return &id
		}
	}
	return nil
}

// resolveUnit prefers the unit recorded at shopID, falling back to the most common
// unit for this name across all shops.
func resolveUnit(group []models.ItemAlias, shopID *uuid.UUID) string {
	if shopID != nil {
		atShop := make([]models.ItemAlias, 0, len(group))
		for _, a := range group {
			if a.ShopID != nil && *a.ShopID == *shopID {
				atShop = append(atShop, a)
			}
		}
		if u := mostCommonUnit(atShop); u != "" {
			return u
		}
	}
	return mostCommonUnit(group)
}

// mostCommonUnit picks the unit backed by the most purchases, breaking ties toward
// the most recently used so the answer is stable rather than map-order dependent.
func mostCommonUnit(group []models.ItemAlias) string {
	type tally struct {
		count int
		best  models.ItemAlias
	}
	tallies := make(map[string]*tally)
	for _, a := range group {
		if a.Unit == "" {
			continue
		}
		t, ok := tallies[a.Unit]
		if !ok {
			tallies[a.Unit] = &tally{count: a.PurchaseCount, best: a}
			continue
		}
		t.count += a.PurchaseCount
		if a.LastUsedAt.After(t.best.LastUsedAt) {
			t.best = a
		}
	}

	var winner string
	var winning *tally
	for unit, t := range tallies {
		switch {
		case winning == nil,
			t.count > winning.count,
			t.count == winning.count && t.best.LastUsedAt.After(winning.best.LastUsedAt):
			winner, winning = unit, t
		}
	}
	return winner
}

// resolveCategory takes the most recently recorded category for this name, breaking
// ties by purchase count. Most-recent rather than most-frequent so that deliberately
// recategorising an item takes effect on the next add instead of being outvoted by
// however many times it was filed the old way.
func resolveCategory(group []models.ItemAlias) *uuid.UUID {
	var best *models.ItemAlias
	for i := range group {
		a := &group[i]
		if a.CategoryID == nil {
			continue
		}
		if best == nil ||
			a.LastUsedAt.After(best.LastUsedAt) ||
			(a.LastUsedAt.Equal(best.LastUsedAt) && a.PurchaseCount > best.PurchaseCount) {
			best = a
		}
	}
	if best == nil {
		return nil
	}
	return best.CategoryID
}
