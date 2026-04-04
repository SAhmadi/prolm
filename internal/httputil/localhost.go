package httputil

import "net/url"

// IsLocalhostURL checks if a URL points to localhost (for test servers).
// Used across packages to bypass HTTPS validation for httptest.Server URLs.
func IsLocalhostURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
