package metrics

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Scraper fetches Prometheus exposition format from collector pods.
type Scraper struct {
	Client *http.Client
}

// NewScraper returns a scraper with the given per-request timeout.
func NewScraper(timeout time.Duration) *Scraper {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &Scraper{
		Client: &http.Client{Timeout: timeout},
	}
}

// neededNames are the raw Prometheus metric names we parse from scrapes.
var neededNames = map[string]struct{}{
	"otelcol_process_memory_rss":              {},
	"otelcol_exporter_queue_size":             {},
	"otelcol_exporter_queue_capacity":         {},
	"otelcol_exporter_enqueue_failed_spans":   {},
	"otelcol_receiver_refused_spans":          {},
}

// ScrapeURL scrapes one endpoint and returns summed samples for needed metrics.
// Label variants are summed (e.g. per-exporter queue size).
func (s *Scraper) ScrapeURL(ctx context.Context, url string) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("scrape %s: status %d: %s", url, resp.StatusCode, string(body))
	}
	return ParsePrometheus(resp.Body)
}

// ScrapeMany scrapes URLs in parallel. Failed scrapes are omitted from results
// (logged by the caller via errors slice).
func (s *Scraper) ScrapeMany(ctx context.Context, urls []string) ([]map[string]float64, []error) {
	out := make([]map[string]float64, 0, len(urls))
	errs := make([]error, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, u := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			samples, err := s.ScrapeURL(ctx, url)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", url, err))
				return
			}
			out = append(out, samples)
		}(u)
	}
	wg.Wait()
	return out, errs
}

// ParsePrometheus parses a subset of Prometheus text exposition format.
func ParsePrometheus(r io.Reader) (map[string]float64, error) {
	sums := make(map[string]float64)
	scanner := bufio.NewScanner(r)
	// Collectors can emit large metric dumps; raise token size.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := parseSampleLine(line)
		if !ok {
			continue
		}
		if _, want := neededNames[name]; !want {
			continue
		}
		sums[name] += value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return sums, nil
}

func parseSampleLine(line string) (name string, value float64, ok bool) {
	// metric{labels} value [timestamp]
	// or metric value
	var rest string
	if i := strings.IndexByte(line, '{'); i >= 0 {
		name = line[:i]
		end := strings.IndexByte(line, '}')
		if end < 0 {
			return "", 0, false
		}
		rest = strings.TrimSpace(line[end+1:])
	} else {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return "", 0, false
		}
		name = fields[0]
		rest = strings.Join(fields[1:], " ")
	}
	fields := strings.Fields(rest)
	if len(fields) < 1 {
		return "", 0, false
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "", 0, false
	}
	return name, v, true
}
