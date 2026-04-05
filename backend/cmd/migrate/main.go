// migrate converts a KinCart SQLite database from uint auto-increment IDs
// to UUID primary keys. Run once before deploying the new auth-v2 backend.
//
// Usage:
//
//	go run ./cmd/migrate --db /path/to/kincart.db
//	go run ./cmd/migrate --db /path/to/kincart.db --dry-run
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := flag.String("db", "", "Path to kincart.db (required)")
	dryRun := flag.Bool("dry-run", false, "Print what would happen without modifying the DB")
	flag.Parse()

	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: migrate --db /path/to/kincart.db [--dry-run]")
		os.Exit(1)
	}

	if _, err := os.Stat(*dbPath); os.IsNotExist(err) {
		log.Fatalf("DB file not found: %s", *dbPath)
	}

	db, err := sql.Open("sqlite3", *dbPath+"?_foreign_keys=off")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if *dryRun {
		fmt.Println("=== DRY RUN — no changes will be written ===")
	}

	m := &migrator{db: db, dryRun: *dryRun}
	if err := m.run(); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	fmt.Println("\nMigration complete.")
}

type migrator struct {
	db     *sql.DB
	dryRun bool

	familyMap   map[int64]string
	userMap     map[int64]string
	listMap     map[int64]string
	categoryMap map[int64]string
	shopMap     map[int64]string
	itemMap     map[int64]string
	receiptMap  map[int64]string
}

func (m *migrator) run() error {
	if err := m.buildMappings(); err != nil {
		return fmt.Errorf("build mappings: %w", err)
	}
	if m.dryRun {
		m.printMappings()
		return nil
	}
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	for _, step := range []func(*sql.Tx) error{
		m.migrateFamilies, m.migrateUsers, m.migrateShoppingLists,
		m.migrateCategories, m.migrateShops, m.migrateItems,
		m.migrateReceipts, m.migrateReceiptItems, m.migrateItemFrequencies,
		m.migrateShopCategoryOrders, m.migrateItemAliases, m.dropAuthTables,
	} {
		if err = step(tx); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (m *migrator) buildMappings() error {
	m.familyMap = make(map[int64]string)
	m.userMap = make(map[int64]string)
	m.listMap = make(map[int64]string)
	m.categoryMap = make(map[int64]string)
	m.shopMap = make(map[int64]string)
	m.itemMap = make(map[int64]string)
	m.receiptMap = make(map[int64]string)

	for _, t := range []struct {
		table string
		m     map[int64]string
	}{
		{"families", m.familyMap}, {"users", m.userMap}, {"shopping_lists", m.listMap},
		{"categories", m.categoryMap}, {"shops", m.shopMap}, {"items", m.itemMap}, {"receipts", m.receiptMap},
	} {
		rows, err := m.db.Query("SELECT id FROM " + t.table)
		if err != nil {
			continue
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return fmt.Errorf("scan %s.id: %w", t.table, err)
			}
			t.m[id] = uuid.New().String()
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (m *migrator) printMappings() {
	fmt.Printf("families: %d | users: %d | lists: %d | categories: %d | shops: %d | items: %d | receipts: %d\n",
		len(m.familyMap), len(m.userMap), len(m.listMap), len(m.categoryMap), len(m.shopMap), len(m.itemMap), len(m.receiptMap))
	for old, new := range m.familyMap {
		fmt.Printf("  family %d → %s\n", old, new)
	}
	for old, new := range m.userMap {
		fmt.Printf("  user %d → %s\n", old, new)
	}
}

func ns(s sql.NullString) interface{} {
	if s.Valid {
		return s.String
	}
	return nil
}
func nf(f sql.NullFloat64) interface{} {
	if f.Valid {
		return f.Float64
	}
	return nil
}
func ni(i sql.NullInt64) interface{} {
	if i.Valid {
		return i.Int64
	}
	return nil
}
func nuuid(m map[int64]string, i sql.NullInt64) interface{} {
	if !i.Valid {
		return nil
	}
	if v, ok := m[i.Int64]; ok {
		return v
	}
	return nil
}

func (m *migrator) migrateFamilies(tx *sql.Tx) error {
	fmt.Println("Migrating families...")
	rows, err := m.db.Query("SELECT id, created_at, updated_at, deleted_at, name, currency FROM families")
	if err != nil {
		return err
	}
	type row struct {
		id        int64
		ca, ua    string
		da        sql.NullString
		name      string
		currency  sql.NullString
	}
	var data []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.ca, &r.ua, &r.da, &r.name, &r.currency); err != nil {
			rows.Close()
			return err
		}
		data = append(data, r)
	}
	rows.Close()
	tx.Exec("DROP TABLE IF EXISTS families")
	tx.Exec(`CREATE TABLE families (id TEXT PRIMARY KEY, created_at datetime, updated_at datetime, deleted_at datetime, name TEXT, currency TEXT)`)
	for _, r := range data {
		nid := m.familyMap[r.id]
		fmt.Printf("  %d → %s (%s)\n", r.id, nid, r.name)
		_, err := tx.Exec("INSERT INTO families VALUES (?,?,?,?,?,?)", nid, r.ca, r.ua, ns(r.da), r.name, ns(r.currency))
		if err != nil {
			return fmt.Errorf("insert family: %w", err)
		}
	}
	return nil
}

func (m *migrator) migrateUsers(tx *sql.Tx) error {
	fmt.Println("Migrating users...")
	rows, err := m.db.Query("SELECT id, created_at, updated_at, deleted_at, username, password_hash, family_id FROM users")
	if err != nil {
		return err
	}
	type row struct {
		id, fid  int64
		ca, ua   string
		da       sql.NullString
		un, pw   string
	}
	var data []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.ca, &r.ua, &r.da, &r.un, &r.pw, &r.fid); err != nil {
			rows.Close()
			return err
		}
		data = append(data, r)
	}
	rows.Close()
	tx.Exec("DROP TABLE IF EXISTS users")
	tx.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, created_at datetime, updated_at datetime, deleted_at datetime, username TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, family_id TEXT NOT NULL)`)
	for _, r := range data {
		nid := m.userMap[r.id]
		fmt.Printf("  %d → %s (%s)\n", r.id, nid, r.un)
		_, err := tx.Exec("INSERT INTO users VALUES (?,?,?,?,?,?,?)", nid, r.ca, r.ua, ns(r.da), r.un, r.pw, m.familyMap[r.fid])
		if err != nil {
			return fmt.Errorf("insert user: %w", err)
		}
	}
	return nil
}

func (m *migrator) migrateShoppingLists(tx *sql.Tx) error {
	fmt.Printf("Migrating shopping_lists (%d)...\n", len(m.listMap))
	rows, err := m.db.Query("SELECT id, created_at, updated_at, deleted_at, family_id, title, status, estimated_amount, actual_amount, completed_at FROM shopping_lists")
	if err != nil {
		return err
	}
	type row struct {
		id, fid       int64
		ca, ua, title string
		da, st, cat   sql.NullString
		ea, aa        sql.NullFloat64
	}
	var data []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.ca, &r.ua, &r.da, &r.fid, &r.title, &r.st, &r.ea, &r.aa, &r.cat); err != nil {
			rows.Close()
			return err
		}
		data = append(data, r)
	}
	rows.Close()
	tx.Exec("DROP TABLE IF EXISTS shopping_lists")
	tx.Exec(`CREATE TABLE shopping_lists (id TEXT PRIMARY KEY, created_at datetime, updated_at datetime, deleted_at datetime, family_id TEXT NOT NULL, title TEXT, status TEXT, estimated_amount REAL, actual_amount REAL, completed_at datetime)`)
	for _, r := range data {
		_, err := tx.Exec("INSERT INTO shopping_lists VALUES (?,?,?,?,?,?,?,?,?,?)", m.listMap[r.id], r.ca, r.ua, ns(r.da), m.familyMap[r.fid], r.title, ns(r.st), nf(r.ea), nf(r.aa), ns(r.cat))
		if err != nil {
			return fmt.Errorf("insert list: %w", err)
		}
	}
	return nil
}

func (m *migrator) migrateCategories(tx *sql.Tx) error {
	fmt.Printf("Migrating categories (%d)...\n", len(m.categoryMap))
	rows, err := m.db.Query("SELECT id, created_at, updated_at, deleted_at, family_id, name, icon, sort_order FROM categories")
	if err != nil {
		return err
	}
	type row struct {
		id, fid       int64
		ca, ua, name  string
		da, icon      sql.NullString
		sortOrder     int
	}
	var data []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.ca, &r.ua, &r.da, &r.fid, &r.name, &r.icon, &r.sortOrder); err != nil {
			rows.Close()
			return err
		}
		data = append(data, r)
	}
	rows.Close()
	tx.Exec("DROP TABLE IF EXISTS categories")
	tx.Exec(`CREATE TABLE categories (id TEXT PRIMARY KEY, created_at datetime, updated_at datetime, deleted_at datetime, family_id TEXT NOT NULL, name TEXT, icon TEXT, sort_order INTEGER)`)
	for _, r := range data {
		_, err := tx.Exec("INSERT INTO categories VALUES (?,?,?,?,?,?,?,?)", m.categoryMap[r.id], r.ca, r.ua, ns(r.da), m.familyMap[r.fid], r.name, ns(r.icon), r.sortOrder)
		if err != nil {
			return fmt.Errorf("insert category: %w", err)
		}
	}
	return nil
}

func (m *migrator) migrateShops(tx *sql.Tx) error {
	fmt.Printf("Migrating shops (%d)...\n", len(m.shopMap))
	rows, err := m.db.Query("SELECT id, created_at, updated_at, deleted_at, family_id, name FROM shops")
	if err != nil {
		return err
	}
	type row struct {
		id, fid      int64
		ca, ua, name string
		da           sql.NullString
	}
	var data []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.ca, &r.ua, &r.da, &r.fid, &r.name); err != nil {
			rows.Close()
			return err
		}
		data = append(data, r)
	}
	rows.Close()
	tx.Exec("DROP TABLE IF EXISTS shops")
	tx.Exec(`CREATE TABLE shops (id TEXT PRIMARY KEY, created_at datetime, updated_at datetime, deleted_at datetime, family_id TEXT NOT NULL, name TEXT)`)
	for _, r := range data {
		_, err := tx.Exec("INSERT INTO shops VALUES (?,?,?,?,?,?)", m.shopMap[r.id], r.ca, r.ua, ns(r.da), m.familyMap[r.fid], r.name)
		if err != nil {
			return fmt.Errorf("insert shop: %w", err)
		}
	}
	return nil
}

func (m *migrator) migrateItems(tx *sql.Tx) error {
	fmt.Printf("Migrating items (%d)...\n", len(m.itemMap))
	rows, err := m.db.Query(`SELECT id, created_at, updated_at, deleted_at, family_id, name, description,
		quantity, unit, is_bought, price, local_photo_path, is_urgent,
		list_id, category_id, flyer_item_id, receipt_item_id FROM items`)
	if err != nil {
		return err
	}
	type row struct {
		id, fid, lid    int64
		catid           sql.NullInt64
		flyerid, recid  sql.NullInt64
		ca, ua, name, unit string
		da, photo, desc sql.NullString
		bought, urgent  bool
		price, qty      sql.NullFloat64
	}
	var data []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.ca, &r.ua, &r.da, &r.fid, &r.name, &r.desc,
			&r.qty, &r.unit, &r.bought, &r.price, &r.photo, &r.urgent,
			&r.lid, &r.catid, &r.flyerid, &r.recid); err != nil {
			rows.Close()
			return fmt.Errorf("scan item: %w", err)
		}
		data = append(data, r)
	}
	rows.Close()
	tx.Exec("DROP TABLE IF EXISTS items")
	tx.Exec(`CREATE TABLE items (id TEXT PRIMARY KEY, created_at datetime, updated_at datetime, deleted_at datetime, family_id TEXT NOT NULL, name TEXT, description TEXT, quantity REAL, unit TEXT, is_bought INTEGER, price REAL, local_photo_path TEXT, is_urgent INTEGER, list_id TEXT NOT NULL, category_id TEXT, flyer_item_id INTEGER, receipt_item_id INTEGER)`)
	for _, r := range data {
		var catID interface{}
		if r.catid.Valid {
			if v, ok := m.categoryMap[r.catid.Int64]; ok {
				catID = v
			}
		}
		_, err := tx.Exec(`INSERT INTO items VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			m.itemMap[r.id], r.ca, r.ua, ns(r.da), m.familyMap[r.fid], r.name, ns(r.desc),
			nf(r.qty), r.unit, r.bought, nf(r.price), ns(r.photo), r.urgent,
			m.listMap[r.lid], catID, ni(r.flyerid), ni(r.recid))
		if err != nil {
			return fmt.Errorf("insert item %d: %w", r.id, err)
		}
	}
	return nil
}

func (m *migrator) migrateReceipts(tx *sql.Tx) error {
	fmt.Printf("Migrating receipts (%d)...\n", len(m.receiptMap))
	rows, err := m.db.Query("SELECT id, created_at, updated_at, deleted_at, family_id, list_id, shop_id, date, total, image_path, status FROM receipts")
	if err != nil {
		return err
	}
	type row struct {
		id, fid         int64
		lid, sid        sql.NullInt64
		ca, ua, imgpath string
		da, date, st    sql.NullString
		total           sql.NullFloat64
	}
	var data []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.ca, &r.ua, &r.da, &r.fid, &r.lid, &r.sid, &r.date, &r.total, &r.imgpath, &r.st); err != nil {
			rows.Close()
			return fmt.Errorf("scan receipt: %w", err)
		}
		data = append(data, r)
	}
	rows.Close()
	tx.Exec("DROP TABLE IF EXISTS receipts")
	tx.Exec(`CREATE TABLE receipts (id TEXT PRIMARY KEY, created_at datetime, updated_at datetime, deleted_at datetime, family_id TEXT NOT NULL, list_id TEXT, shop_id TEXT, date datetime, total REAL, image_path TEXT, status TEXT)`)
	for _, r := range data {
		_, err := tx.Exec("INSERT INTO receipts VALUES (?,?,?,?,?,?,?,?,?,?,?)",
			m.receiptMap[r.id], r.ca, r.ua, ns(r.da), m.familyMap[r.fid],
			nuuid(m.listMap, r.lid), nuuid(m.shopMap, r.sid), ns(r.date), nf(r.total), r.imgpath, ns(r.st))
		if err != nil {
			return fmt.Errorf("insert receipt: %w", err)
		}
	}
	return nil
}

func (m *migrator) migrateReceiptItems(tx *sql.Tx) error {
	fmt.Println("Migrating receipt_items...")
	rows, err := m.db.Query(`SELECT id, receipt_id, name, quantity, unit, price, total_price,
		matched_item_id, match_status, confidence, suggested_items FROM receipt_items`)
	if err != nil {
		return err
	}
	type row struct {
		id, rid        int64
		mid            sql.NullInt64
		name, unit     string
		ms, si         sql.NullString
		qty, price, tp float64
		conf           int
	}
	var data []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.rid, &r.name, &r.qty, &r.unit, &r.price, &r.tp, &r.mid, &r.ms, &r.conf, &r.si); err != nil {
			rows.Close()
			return fmt.Errorf("scan receipt_item: %w", err)
		}
		data = append(data, r)
	}
	rows.Close()
	tx.Exec("DROP TABLE IF EXISTS receipt_items")
	tx.Exec(`CREATE TABLE receipt_items (id INTEGER PRIMARY KEY AUTOINCREMENT, receipt_id TEXT NOT NULL, name TEXT, quantity REAL, unit TEXT, price REAL, total_price REAL, matched_item_id TEXT, match_status TEXT, confidence INTEGER, suggested_items TEXT)`)
	for _, r := range data {
		_, err := tx.Exec(`INSERT INTO receipt_items(id,receipt_id,name,quantity,unit,price,total_price,matched_item_id,match_status,confidence,suggested_items) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			r.id, m.receiptMap[r.rid], r.name, r.qty, r.unit, r.price, r.tp, nuuid(m.itemMap, r.mid), ns(r.ms), r.conf, ns(r.si))
		if err != nil {
			return fmt.Errorf("insert receipt_item: %w", err)
		}
	}
	return nil
}

func (m *migrator) migrateItemFrequencies(tx *sql.Tx) error {
	fmt.Println("Migrating item_frequencies...")
	rows, err := m.db.Query("SELECT id, family_id, item_name, frequency, last_price FROM item_frequencies")
	if err != nil {
		return err
	}
	type row struct {
		id, fid   int64
		name      string
		freq      int
		lp        sql.NullFloat64
	}
	var data []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.fid, &r.name, &r.freq, &r.lp); err != nil {
			rows.Close()
			return err
		}
		data = append(data, r)
	}
	rows.Close()
	tx.Exec("DROP TABLE IF EXISTS item_frequencies")
	tx.Exec(`CREATE TABLE item_frequencies (id INTEGER PRIMARY KEY AUTOINCREMENT, family_id TEXT NOT NULL, item_name TEXT, frequency INTEGER, last_price REAL)`)
	for _, r := range data {
		_, err := tx.Exec("INSERT INTO item_frequencies(id,family_id,item_name,frequency,last_price) VALUES (?,?,?,?,?)", r.id, m.familyMap[r.fid], r.name, r.freq, nf(r.lp))
		if err != nil {
			return fmt.Errorf("insert item_freq: %w", err)
		}
	}
	return nil
}

func (m *migrator) migrateShopCategoryOrders(tx *sql.Tx) error {
	fmt.Println("Migrating shop_category_orders...")
	rows, err := m.db.Query("SELECT id, shop_id, category_id, sort_order FROM shop_category_orders")
	if err != nil {
		return err
	}
	type row struct {
		id, sid, cid int64
		sortOrder    int
	}
	var data []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.sid, &r.cid, &r.sortOrder); err != nil {
			rows.Close()
			return err
		}
		data = append(data, r)
	}
	rows.Close()
	tx.Exec("DROP TABLE IF EXISTS shop_category_orders")
	tx.Exec(`CREATE TABLE shop_category_orders (id INTEGER PRIMARY KEY AUTOINCREMENT, shop_id TEXT NOT NULL, category_id TEXT NOT NULL, sort_order INTEGER)`)
	for _, r := range data {
		sid, cid := m.shopMap[r.sid], m.categoryMap[r.cid]
		if sid == "" || cid == "" {
			fmt.Printf("  WARNING: skipping shop_category_order %d\n", r.id)
			continue
		}
		_, err := tx.Exec("INSERT INTO shop_category_orders(id,shop_id,category_id,sort_order) VALUES (?,?,?,?)", r.id, sid, cid, r.sortOrder)
		if err != nil {
			return fmt.Errorf("insert sco: %w", err)
		}
	}
	return nil
}

func (m *migrator) migrateItemAliases(tx *sql.Tx) error {
	fmt.Println("Migrating item_aliases...")
	rows, err := m.db.Query(`SELECT id, family_id, planned_name, receipt_name, shop_id, last_price, purchase_count, last_used_at, created_at FROM item_aliases`)
	if err != nil {
		fmt.Println("  skipping item_aliases (not found or incompatible)")
		return nil
	}
	type row struct {
		id, fid      int64
		sid          sql.NullInt64
		pn, rn       string
		lp           sql.NullFloat64
		pc           int
		lua, ca      sql.NullString
	}
	var data []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.fid, &r.pn, &r.rn, &r.sid, &r.lp, &r.pc, &r.lua, &r.ca); err != nil {
			rows.Close()
			return fmt.Errorf("scan alias: %w", err)
		}
		data = append(data, r)
	}
	rows.Close()
	tx.Exec("DROP TABLE IF EXISTS item_aliases")
	tx.Exec(`CREATE TABLE item_aliases (id INTEGER PRIMARY KEY AUTOINCREMENT, family_id TEXT NOT NULL, planned_name TEXT NOT NULL, receipt_name TEXT NOT NULL, shop_id TEXT, last_price REAL, purchase_count INTEGER, last_used_at datetime, created_at datetime)`)
	for _, r := range data {
		_, err := tx.Exec(`INSERT INTO item_aliases(id,family_id,planned_name,receipt_name,shop_id,last_price,purchase_count,last_used_at,created_at) VALUES (?,?,?,?,?,?,?,?,?)`,
			r.id, m.familyMap[r.fid], r.pn, r.rn, nuuid(m.shopMap, r.sid), nf(r.lp), r.pc, ns(r.lua), ns(r.ca))
		if err != nil {
			return fmt.Errorf("insert alias: %w", err)
		}
	}
	return nil
}

func (m *migrator) dropAuthTables(tx *sql.Tx) error {
	fmt.Println("Dropping old auth tables (recreated by AutoMigrate)...")
	tx.Exec("DROP TABLE IF EXISTS refresh_tokens")
	tx.Exec("DROP TABLE IF EXISTS blacklisted_tokens")
	return nil
}
