package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"kincart/internal/flyers"
	"kincart/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/subosito/gotenv"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	var filePath string
	var port int
	var serveOnly bool
	var shopName string
	flag.StringVar(&filePath, "file", "", "Path to local flyer file (image/PDF)")
	flag.StringVar(&shopName, "shop", "", "Name of the shop (mandatory for parsing)")
	flag.IntVar(&port, "port", 8081, "Port for the HTTP server")
	flag.BoolVar(&serveOnly, "serve", false, "Start only the HTTP server to view previous results")
	flag.Parse()

	_ = gotenv.Load()

	dbPath := "temp_flyers.db"
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to temp database: %v", err)
	}

	// Auto migrate temp db
	if err := db.AutoMigrate(&models.Flyer{}, &models.FlyerItem{}); err != nil {
		log.Fatalf("failed to migrate temp database: %v", err)
	}

	if serveOnly {
		startServer(db, port)
		return
	}

	if !serveOnly && filePath != "" && shopName == "" {
		log.Fatal("-shop is mandatory when parsing a file")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY must be set in .env")
	}

	parser, err := flyers.NewParser(apiKey)
	if err != nil {
		log.Fatalf("failed to create parser: %v", err)
	}

	manager := flyers.NewManager(db, parser)

	if filePath != "" {
		parseLocalFile(manager, parser, filePath, shopName)
	}

	startServer(db, port)
}

func parseLocalFile(manager *flyers.Manager, parser *flyers.Parser, path string, shopName string) {
	fmt.Printf("Parsing local file: %s\n", path)
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	ext := filepath.Ext(path)
	mime := "image/jpeg"
	if ext == ".png" {
		mime = "image/png"
	} else if ext == ".pdf" {
		mime = "application/pdf"
	}

	tempDir, err := os.MkdirTemp("", "flyer-parse-cli-*")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	var pages []flyers.Attachment
	if mime == "application/pdf" {
		fmt.Println("Splitting PDF into pages...")
		pageFiles, err := flyers.SplitPDF(data, tempDir)
		if err != nil {
			log.Fatalf("failed to split PDF: %v", err)
		}
		for _, pf := range pageFiles {
			pData, err := os.ReadFile(pf)
			if err != nil {
				continue
			}
			pages = append(pages, flyers.Attachment{
				Filename:    filepath.Base(pf),
				ContentType: "image/png",
				Data:        pData,
			})
		}
	} else {
		pages = append(pages, flyers.Attachment{
			Filename:    filepath.Base(path),
			ContentType: mime,
			Data:        data,
		})
	}

	ctx := context.Background()
	total := len(pages)
	for i, p := range pages {
		fmt.Printf("[%d/%d] Parsing page: %s...\n", i+1, total, p.Filename)
		parsed, err := parser.ParseFlyer(ctx, []flyers.Attachment{p})
		if err != nil {
			slog.Error("failed to parse page", "current", i+1, "total", total, "file", p.Filename, "err", err)
			continue
		}

		fmt.Printf("[%d/%d] Saving items from %s: %s...\n", i+1, total, p.Filename, shopName)
		if err := manager.SaveParsedFlyer(parsed, p.Data, shopName); err != nil {
			slog.Error("failed to save page", "current", i+1, "total", total, "file", p.Filename, "err", err)
			continue
		}
	}

	fmt.Printf("\nProcessing completed! Successfully processed %d pages.\n", total)
}

func startServer(db *gorm.DB, port int) {
	r := gin.Default()

	// Serve cropped images
	r.Static("/data/flyer_items", "./data/flyer_items")

	r.SetHTMLTemplate(loadTemplate())

	r.GET("/", func(c *gin.Context) {
		q := c.Query("q")
		var flyers []models.Flyer

		query := db.Order("created_at desc")
		if q != "" {
			term := "%" + q + "%"
			// Only show flyers that have matching items
			query = query.Where("id IN (SELECT flyer_id FROM flyer_items WHERE name LIKE ? OR categories LIKE ? OR keywords LIKE ?)", term, term, term)
			// Filter items within the flyer record
			query = query.Preload("Items", "name LIKE ? OR categories LIKE ? OR keywords LIKE ?", term, term, term)
		} else {
			query = query.Preload("Items")
		}

		if err := query.Find(&flyers).Error; err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.HTML(http.StatusOK, "index.html", gin.H{
			"flyers": flyers,
			"query":  q,
		})
	})

	fmt.Printf("Web UI available at http://localhost:%d\n", port)
	r.Run(fmt.Sprintf(":%d", port))
}

func loadTemplate() *template.Template {
	html := `
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Parsed Flyers</title>
			<style>
				body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; margin: 40px; background: #f0f2f5; color: #1a1a1a; }
				.container { max-width: 1200px; margin: 0 auto; }
				.flyer { background: white; padding: 30px; margin-bottom: 40px; border-radius: 12px; box-shadow: 0 4px 6px rgba(0,0,0,0.05); }
				.flyer h2 { margin-top: 0; color: #000; border-bottom: 2px solid #f0f2f5; padding-bottom: 10px; }
				.meta { color: #65676b; margin-bottom: 20px; font-size: 0.9em; }
				.items { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 25px; }
				.item { background: #fff; border: 1px solid #e4e6eb; padding: 15px; border-radius: 8px; transition: transform 0.2s; }
				.item:hover { transform: translateY(-5px); box-shadow: 0 8px 15px rgba(0,0,0,0.1); }
				.item img { width: 100%; height: 200px; object-fit: contain; background: #fafafa; border-radius: 6px; margin-bottom: 12px; }
				.item .name { font-weight: 600; font-size: 1.1em; margin-bottom: 8px; height: 2.4em; overflow: hidden; }
				.item .price-box { display: flex; justify-content: space-between; align-items: center; }
				.item .price { font-size: 1.4em; font-weight: bold; color: #00a884; }
				.item .quantity { color: #65676b; font-size: 0.9em; }
				h1 { color: #1c1e21; margin-bottom: 10px; text-align: center; }
				.search-box { text-align: center; margin-bottom: 30px; }
				.search-box input { padding: 10px 15px; width: 400px; border: 1px solid #e4e6eb; border-radius: 20px; outline: none; box-shadow: 0 2px 4px rgba(0,0,0,0.05); }
				.search-box button { padding: 10px 20px; background: #00a884; color: white; border: none; border-radius: 20px; cursor: pointer; margin-left: 10px; font-weight: 600; }
				.search-box a { color: #65676b; text-decoration: none; font-size: 0.9em; margin-left: 10px; }
			</style>
		</head>
		<body>
			<div class="container">
				<h1>Flyer Parsing Results</h1>
				<div class="search-box">
					<form action="/" method="GET">
						<input type="text" name="q" value="{{.query}}" placeholder="Search items by name, category, or keywords...">
						<button type="submit">Filter</button>
						{{if .query}}<a href="/">Clear Filter</a>{{end}}
					</form>
				</div>
				{{range .flyers}}
				<div class="flyer">
					<h2>{{.ShopName}}</h2>
					<div class="meta">
						Validity: {{.StartDate.Format "2006-01-02"}} to {{.EndDate.Format "2006-01-02"}}<br>
						Processed: {{.CreatedAt.Format "2006-01-02 15:04"}}
					</div>
					<div class="items">
						{{range .Items}}
						<div class="item">
							{{if .LocalPhotoPath}}
							<img src="/{{.LocalPhotoPath}}" alt="{{.Name}}">
							{{else}}
							<div style="height:200px; display:flex; align-items:center; justify-content:center; background:#eee; color:#999; border-radius:6px; margin-bottom:12px;">No Image</div>
							{{end}}
							<div class="name">{{.Name}}</div>
							<div class="price-box">
								<div>
									<div class="price">{{.Price}}</div>
									{{if .OriginalPrice}}
									<div style="text-decoration: line-through; color: #65676b; font-size: 0.9em;">{{.OriginalPrice}}</div>
									{{end}}
								</div>
								<div class="quantity">{{.Quantity}}</div>
							</div>
							<div style="margin-top: 5px; font-size: 0.8em; color: #65676b;">
								Validity: {{.StartDate.Format "2006-01-02"}} to {{.EndDate.Format "2006-01-02"}}
							</div>
							<div style="margin-top: 10px; font-size: 0.8em; color: #65676b;">
								{{if .Categories}}<div><strong>Categories:</strong> {{.Categories}}</div>{{end}}
								{{if .Keywords}}<div><strong>Keywords:</strong> {{.Keywords}}</div>{{end}}
							</div>
						</div>
						{{end}}
					</div>
				</div>
				{{else}}
				<div style="text-align:center; padding: 50px; background:white; border-radius:12px;">
					<p>No flyers parsed yet. Run with <code>-file path/to/image.jpg</code> to parse your first flyer.</p>
				</div>
				{{end}}
			</div>
		</body>
		</html>
		`
	return template.Must(template.New("index.html").Parse(html))
}
