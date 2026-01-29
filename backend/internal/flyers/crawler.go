package flyers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const BaseURL = "https://www.akcniceny.cz"

type RetailerConfig struct {
	URL    string
	Filter func(string) bool
}

var Retailers = map[string]RetailerConfig{
	"albert": {
		URL:    fmt.Sprintf("%s/letaky/albert/kraj-praha/praha/", BaseURL),
		Filter: func(title string) bool { return strings.Contains(strings.ToLower(title), "hypermarket") },
	},
	"billa": {
		URL:    fmt.Sprintf("%s/letaky/billa/", BaseURL),
		Filter: func(title string) bool { return !strings.Contains(strings.ToLower(title), "malý leták") },
	},
	"tesco": {
		URL:    fmt.Sprintf("%s/letaky/tesco/", BaseURL),
		Filter: func(title string) bool { return strings.Contains(strings.ToLower(title), "hypermarkety") },
	},
	"kaufland": {
		URL:    fmt.Sprintf("%s/letaky/kaufland/kraj-praha/praha/kaufland-praha-5-stodulky-pod-hranici-1304-17/", BaseURL),
		Filter: func(title string) bool { return !strings.Contains(strings.ToLower(title), "spotřební zboží") },
	},
	"globus": {
		URL:    fmt.Sprintf("%s/letaky/globus/kraj-praha/praha/globus-hypermarket-a-baumarkt-praha-zlicin-sarska-5133-praha-5/", BaseURL),
		Filter: func(_ string) bool { return true },
	},
	"lidl": {
		URL:    fmt.Sprintf("%s/letaky/lidl/", BaseURL),
		Filter: func(_ string) bool { return true },
	},
}

type FlyerInfo struct {
	ID    string
	URL   string
	Title string
}

type Crawler struct {
	client *http.Client
}

func NewCrawler() *Crawler {
	return &Crawler{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Crawler) FetchFlyerURLs(retailerName string) ([]FlyerInfo, error) {
	config, ok := Retailers[retailerName]
	if !ok {
		return nil, fmt.Errorf("unknown retailer: %s", retailerName)
	}

	resp, err := c.client.Get(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch retailer page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	html := string(body)
	blocks := strings.Split(html, `itemtype="http://schema.org/SaleEvent"`)
	if len(blocks) < 2 {
		return nil, nil
	}

	nameRegex := regexp.MustCompile(`itemprop="name" content="([^"]+)"`)
	urlRegex := regexp.MustCompile(`itemprop="url" content="([^"]+)"`)

	var flyers []FlyerInfo
	for _, block := range blocks[1:] {
		nameMatch := nameRegex.FindStringSubmatch(block)
		urlMatch := urlRegex.FindStringSubmatch(block)

		if len(nameMatch) > 1 && len(urlMatch) > 1 {
			title := strings.TrimSpace(nameMatch[1])
			fullURL := strings.TrimSpace(urlMatch[1])

			// Extract ID
			relativeURL := strings.Replace(fullURL, BaseURL, "", 1)
			parts := strings.Split(relativeURL, "-")
			idPart := parts[len(parts)-1]
			id := strings.Split(idPart, "/")[0]

			if config.Filter(title) {
				flyers = append(flyers, FlyerInfo{
					ID:    id,
					URL:   fullURL,
					Title: title,
				})
			}
		}
	}

	return flyers, nil
}

func (c *Crawler) FetchFlyerImages(flyerURL string, delay time.Duration) ([]string, error) {
	var images []string
	currentURL := flyerURL

	imgRegex := regexp.MustCompile(`src="(https://[^"]+staticac\.cz/foto/letaky/[^"]+\.jpg)"`)
	nextPageRegex := regexp.MustCompile(`href="(/letak/[^/]+/strana-\d+/)"[^>]*>Zobrazit další stránky</a>`)
	paginationRegex := regexp.MustCompile(`href="(/letak/[^/]+/strana-\d+/)"[^>]*aria-label="Následující"`)

	seenImages := make(map[string]bool)

	for currentURL != "" {
		resp, err := c.client.Get(currentURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch flyer page: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read flyer body: %w", err)
		}

		html := string(body)

		matches := imgRegex.FindAllStringSubmatch(html, -1)
		for _, m := range matches {
			if len(m) > 1 {
				imgURL := m[1]
				if !seenImages[imgURL] {
					images = append(images, imgURL)
					seenImages[imgURL] = true
				}
			}
		}

		nextURL := ""
		if match := nextPageRegex.FindStringSubmatch(html); len(match) > 1 {
			nextURL = BaseURL + match[1]
		} else if match := paginationRegex.FindStringSubmatch(html); len(match) > 1 {
			nextURL = BaseURL + match[1]
		}

		if nextURL == "" || nextURL == currentURL {
			break
		}

		currentURL = nextURL
		if delay > 0 {
			time.Sleep(delay)
		}
	}

	return images, nil
}

func (c *Crawler) DownloadImage(url, destPath string) error {
	resp, err := c.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	if err = os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
