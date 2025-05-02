package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os/exec"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

const seenFile = "seen.json"

// loadSeen reads seen IDs from seen.json (or returns empty map)
func loadSeen() map[string]bool {
	seen := make(map[string]bool)
	if data, err := ioutil.ReadFile(seenFile); err == nil {
		json.Unmarshal(data, &seen)
	}
	return seen
}

// saveSeen writes the seen-IDs map back to seen.json
func saveSeen(seen map[string]bool) {
	data, _ := json.MarshalIndent(seen, "", "  ")
	ioutil.WriteFile(seenFile, data, 0644)
}

// fetchListings uses chromedp to render the page and scrape data-obid attributes
func fetchListings(ctx context.Context, pageURL string) ([]string, error) {
	// Navigate + wait for network idle
	var nodes []*cdp.Node
	if err := chromedp.Run(ctx,
		chromedp.Navigate(pageURL),
		chromedp.Sleep(3*time.Second), // give JS time to load
		chromedp.Nodes(`div.listing-card[data-obid]`, &nodes, chromedp.ByQueryAll),
	); err != nil {
		return nil, err
	}

	// Extract each data-obid
	var ids []string
	for _, n := range nodes {
		for i := 0; i+1 < len(n.Attributes); i += 2 {
			if n.Attributes[i] == "data-obid" {
				ids = append(ids, n.Attributes[i+1])
			}
		}
	}
	return ids, nil
}

// notifyNew fires a Linux desktop notification
func notifyNew(obid string) error {
	title := "New flat listing"
	body := "ID: " + obid
	cmd := exec.Command("notify-send", title, body)
	return cmd.Run()
}

// checkOnce scrapes, notifies on unseen, and updates the seen map
func checkOnce(ctx context.Context, url string, seen map[string]bool) {
	ids, err := fetchListings(ctx, url)
	if err != nil {
		log.Println("Fetch error:", err)
		return
	}
	for _, id := range ids {
		if !seen[id] {
			log.Println("→ New listing found:", id)
			if err := notifyNew(id); err != nil {
				log.Println("Notify error:", err)
			} else {
				seen[id] = true
			}
		}
	}
	saveSeen(seen)
}

func main() {
	pageURL := "https://www.immobilienscout24.de/Suche/shape/wohnung-mieten?shape=b2FrX0lzd2NwQXB2QX12QWJyRGdsQXBFa3JLZXtAdXtDcVZ7aEZzYUFtRWliQHJWZ3NBdG5AfXNAbntFcWVBY0FraUNuc1F6ZEBkUHxvQWFOcHRAYE47YXF1X0ltY2pwQXRqQWFrRnRsQG1jQGtNX2ZCcmFBcV9CflxlZkNsa0BxX0J8YEB3WHRfQHtyQndsQH1nQV9uQGthQGJIY3FKfW9EfnZBX2hCdn5OYXVBeGJIZkBicURgakBkd0I.&numberofrooms=4.0-&price=1000.0-2200.0&livingspace=70.0-&exclusioncriteria=swapflat&pricetype=rentpermonth&saveSearchId=137791090&sorting=2"
	intervalMin := 5

	// Ensure notify-send is installed
	if _, err := exec.LookPath("notify-send"); err != nil {
		log.Fatal("notify-send not found; install libnotify-bin")
	}

	// Set up a headless Chrome instance
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	// (Optional) increase PDFDOM complexity limits, timeouts, etc.

	// Prime the session by loading the homepage once (to get cookies/consent)
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.immobilienscout24.de/"),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		log.Println("Warning: could not prime session:", err)
	}

	seen := loadSeen()
	ticker := time.NewTicker(time.Duration(intervalMin) * time.Minute)
	defer ticker.Stop()

	log.Printf("Starting browser-based poller: every %d minutes → %s\n", intervalMin, pageURL)
	checkOnce(ctx, pageURL, seen) // run immediately

	for range ticker.C {
		checkOnce(ctx, pageURL, seen)
	}
}
