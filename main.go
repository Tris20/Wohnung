package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
)

const seenFile = "seen.json"

func loadSeen() map[string]bool {
	m := map[string]bool{}
	if b, err := ioutil.ReadFile(seenFile); err == nil {
		json.Unmarshal(b, &m)
		log.Printf("[DEBUG] Loaded %d seen IDs\n", len(m))
	} else {
		log.Printf("[DEBUG] No seen.json, starting fresh\n")
	}
	return m
}

func saveSeen(m map[string]bool) {
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := ioutil.WriteFile(seenFile, b, 0644); err != nil {
		log.Printf("[ERROR] writing %s: %v\n", seenFile, err)
	} else {
		log.Printf("[DEBUG] Persisted %d seen IDs\n", len(m))
	}
}

func notifySend(id string) {
	title := fmt.Sprintf("New flat listing: %s", id)
	body := time.Now().Format("15:04:05 — ID: " + id)

	log.Printf("[DEBUG] notifySend → %q | %q", title, body)
	cmd := exec.Command(
		"notify-send",
		"-u", "critical",
		title,
		body,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[ERROR] notify-send failed: %v — output: %q", err, out)
	} else {
		log.Println("[DEBUG] notify-send succeeded")
	}
	time.Sleep(200 * time.Millisecond)
}

func main() {
	// 1) Your search URL:
	pageURL := "https://www.immobilienscout24.de/Suche/shape/wohnung-mieten?shape=b2FrX0lzd2NwQXB2QX12QWJyRGdsQXBFa3JLZXtAdXtDcVZ7aEZzYUFtRWliQHJWZ3NBdG5AfXNAbntFcWVBY0FraUNuc1F6ZEBkUHxvQWFOcHRAYE47YXF1X0ltY2pwQXRqQWFrRnRsQG1jQGtNX2ZCcmFBcV9CflxlZkNsa0BxX0J8YEB3WHRfQHtyQndsQH1nQV9uQGthQGJIY3FKfW9EfnZBX2hCdn5OYXVBeGJIZkBicURgakBkd0I.&numberofrooms=1.0-&price=100.0-2200.0&livingspace=1.0-&exclusioncriteria=swapflat&pricetype=rentpermonth&sorting=2"

	pageURL = "https://www.immobilienscout24.de/Suche/shape/wohnung-mieten?shape=b2FrX0lzd2NwQXB2QX12QWJyRGdsQXBFa3JLZXtAdXtDcVZ7aEZzYUFtRWliQHJWZ3NBdG5AfXNAbntFcWVBY0FraUNuc1F6ZEBkUHxvQWFOcHRAYE47YXF1X0ltY2pwQXRqQWFrRnRsQG1jQGtNX2ZCcmFBcV9CflxlZkNsa0BxX0J8YEB3WHRfQHtyQndsQH1nQV9uQGthQGJIY3FKfW9EfnZBX2hCdn5OYXVBeGJIZkBicURgakBkd0I.&numberofrooms=4.0-&price=1000.0-2200.0&livingspace=70.0-&exclusioncriteria=swapflat&pricetype=rentpermonth&saveSearchId=137791090&sorting=2"

	// 2) Launch headed Chrome via Rod + Launcher
	u := launcher.New().
		Headless(false). // ← disable headless
		Devtools(false). // ← or true if you want DevTools
		//Add any flags you like:
		// Set("start-maximized", "").
		MustLaunch()

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	// 3) Stealth to further reduce detection
	page := stealth.MustPage(browser)

	// 4) Maximize so you see everything
	page.MustWindowMaximize()

	seen := loadSeen()
	poll := 5 * time.Minute

	log.Printf("→ Opening browser and navigating to search page…\n")
	page.MustNavigate(pageURL).MustWaitLoad()

	log.Printf("→ Starting polling loop every %v\n", poll)
	for {
		// 5) Scrape current IDs
		els, err := page.Elements("div.listing-card[data-obid]")
		if err != nil {
			log.Printf("[ERROR] selecting cards: %v\n", err)
		} else {
			log.Printf("[DEBUG] Found %d cards\n", len(els))
			for _, el := range els {
				if attr, _ := el.Attribute("data-obid"); attr != nil {
					id := *attr
					if !seen[id] {
						log.Printf("[INFO] New listing: %s\n", id)
						notifySend(id)
						seen[id] = true
					}
				}
			}
			saveSeen(seen)
		}

		// 6) Wait, then reload
		log.Printf("→ Sleeping %v then reloading…\n", poll)
		time.Sleep(poll)
		page.MustReload().MustWaitLoad()
	}
}
