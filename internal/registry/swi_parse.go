package registry

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// parsePackList parses the SWI pack list page (/pack/list) and extracts
// package names, latest versions, and descriptions from the HTML table.
func parsePackList(r io.Reader) ([]PackageVersion, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, &ErrInvalidResponse{Reason: fmt.Sprintf("HTML parse error: %v", err)}
	}

	var results []PackageVersion

	// Walk the DOM looking for table rows with pack data.
	// The pack list page has a table with columns: Name, Version, Downloads, Rating, Description.
	// Pack names are in <a> tags linking to /pack/list?p=<name>.
	walkDOM(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "tr" {
			if pv, ok := parsePackListRow(n); ok {
				if err := validatePackageName(pv.Name); err != nil {
					// Skip entries with invalid names rather than failing entirely (SEC-9).
					return true
				}
				results = append(results, pv)
				return true // don't recurse into matched <tr>
			}
		}
		return false
	})

	return results, nil
}

// parsePackListRow attempts to extract a PackageVersion from a <tr> element.
// Returns (pv, true) if the row contains pack data, (zero, false) otherwise.
func parsePackListRow(tr *html.Node) (PackageVersion, bool) {
	var cells []string
	var packName string

	for td := tr.FirstChild; td != nil; td = td.NextSibling {
		if td.Type != html.ElementNode || (td.Data != "td" && td.Data != "th") {
			continue
		}
		text := extractText(td)
		cells = append(cells, strings.TrimSpace(text))

		// Look for <a> tag with href containing /pack/list?p= to get pack name.
		if packName == "" {
			packName = findPackLink(td)
		}
	}

	// Need at least name and version columns.
	if packName == "" || len(cells) < 2 {
		return PackageVersion{}, false
	}

	pv := PackageVersion{
		Name:    packName,
		Version: normalizeVersion(cells[1]),
	}
	return pv, true
}

// parsePackDetail parses an individual pack detail page (/pack/list?p=<name>)
// and extracts all versions with their download URLs, checksums, and dependencies.
func parsePackDetail(r io.Reader, name string) ([]PackageVersion, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, &ErrInvalidResponse{Reason: fmt.Sprintf("HTML parse error: %v", err)}
	}

	// Check for "not found" page by looking for SWI-Prolog-specific phrases.
	if containsNotFoundText(doc) {
		return nil, &ErrPackageNotFound{Name: name, Registry: "swi-pack-index"}
	}

	var versions []PackageVersion
	tables := findElements(doc, "table")

	for _, table := range tables {
		rows := findElements(table, "tr")
		for _, row := range rows {
			if pv, ok := parseDetailRow(row, name); ok {
				versions = append(versions, pv)
			}
		}
	}

	return versions, nil
}

// parseDetailRow extracts version info from a detail page table row.
// Expects columns like: Version, SHA1/SHA256, #Downloads, URL.
func parseDetailRow(tr *html.Node, name string) (PackageVersion, bool) {
	var cells []string
	var links []string

	for td := tr.FirstChild; td != nil; td = td.NextSibling {
		if td.Type != html.ElementNode || (td.Data != "td" && td.Data != "th") {
			continue
		}
		text := strings.TrimSpace(extractText(td))
		cells = append(cells, text)

		// Collect all href links in this cell.
		links = append(links, findHrefs(td)...)
	}

	// Skip header rows and rows without enough data.
	if len(cells) < 2 {
		return PackageVersion{}, false
	}

	// Try to identify version string in first cell.
	version := normalizeVersion(cells[0])
	if version == "" {
		return PackageVersion{}, false
	}

	pv := PackageVersion{
		Name:    name,
		Version: version,
	}

	// Look for SHA1 hash in cells.
	for _, cell := range cells[1:] {
		if validateSHA1(strings.ToLower(cell)) {
			pv.Checksum = "sha1:" + strings.ToLower(cell)
			break
		}
	}

	// Use the first valid download link.
	for _, href := range links {
		if isDownloadURL(href) {
			pv.URL = href
			break
		}
		if archiveURL, ok := githubArchiveFromGitURL(href, version); ok {
			pv.URL = archiveURL
			// SWI pack pages report SHA1 for their own fetch path (often .git),
			// which does not necessarily match bytes from GitHub archive URLs.
			// Keep lockfile integrity via computed sha256 after download.
			pv.Checksum = ""
			pv.ChecksumWarning = fmt.Sprintf(
				"upstream checksum for %s@%s could not be cross-verified against the archive URL; proceeding with lockfile sha256 pinning",
				name, version,
			)
			break
		}
	}

	return pv, true
}

// walkDOM recursively visits every node in the tree rooted at n.
// If fn returns true the node's children are skipped.
func walkDOM(n *html.Node, fn func(*html.Node) bool) {
	if fn(n) {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkDOM(c, fn)
	}
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c))
	}
	return sb.String()
}

func findPackLink(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				if name := extractPackNameFromHref(attr.Val); name != "" {
					return name
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if name := findPackLink(c); name != "" {
			return name
		}
	}
	return ""
}

func extractPackNameFromHref(href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if !strings.HasPrefix(u.Path, "/pack/list") {
		return ""
	}
	return u.Query().Get("p")
}

func findHrefs(n *html.Node) []string {
	var hrefs []string
	walkDOM(n, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					hrefs = append(hrefs, attr.Val)
				}
			}
		}
		return false
	})
	return hrefs
}

func findElements(n *html.Node, tag string) []*html.Node {
	var result []*html.Node
	walkDOM(n, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == tag {
			result = append(result, node)
		}
		return false
	})
	return result
}

func containsNotFoundText(doc *html.Node) bool {
	text := strings.ToLower(extractText(doc))
	return strings.Contains(text, "no packs match") ||
		strings.Contains(text, "know nothing about")
}

func normalizeVersion(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return ""
	}
	if s[0] < '0' || s[0] > '9' {
		return ""
	}
	return s
}

func isDownloadURL(href string) bool {
	lower := strings.ToLower(href)
	return strings.HasSuffix(lower, ".tgz") ||
		strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".zip")
}

func githubArchiveFromGitURL(href, version string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(href))
	if !strings.HasSuffix(lower, ".git") {
		return "", false
	}

	u, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	if !strings.EqualFold(u.Hostname(), "github.com") {
		return "", false
	}

	parts := splitPathParts(u.Path)
	if len(parts) < 2 {
		return "", false
	}
	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], ".git")
	if owner == "" || repo == "" {
		return "", false
	}

	// Prefer the common GitHub tag convention used by SWI pack pages.
	return fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/v%s.tar.gz", owner, repo, version), true
}

func splitPathParts(p string) []string {
	rawParts := strings.Split(strings.Trim(p, "/"), "/")
	out := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
