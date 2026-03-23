package monitors

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/statusy/statusy/internal/models"
)

// RunCheck dispatches to the appropriate checker based on monitor type.
func RunCheck(ctx context.Context, m *models.Monitor) (models.MonitorStatus, string) {
	switch m.Type {
	case models.MonitorHTTP:
		return checkHTTP(ctx, m)
	case models.MonitorPort:
		return checkPort(ctx, m)
	case models.MonitorPing:
		return checkPing(ctx, m)
	case models.MonitorKeyword:
		return checkKeyword(ctx, m)
	case models.MonitorJSONAPI:
		return checkJSONAPI(ctx, m)
	case models.MonitorUDP:
		return checkUDP(ctx, m)
	case models.MonitorResponseTime:
		return checkResponseTime(ctx, m)
	case models.MonitorDNS:
		return checkDNS(ctx, m)
	case models.MonitorSSL:
		return checkSSL(ctx, m)
	case models.MonitorDomainExpiry:
		return checkDomainExpiry(ctx, m)
	default:
		return models.StatusDown, fmt.Sprintf("unknown monitor type: %s", m.Type)
	}
}

// ─── HTTP ─────────────────────────────────────────────────────────────────────

// tlsCertExpiry dials the HTTPS host and returns the leaf certificate expiry time.
func tlsCertExpiry(ctx context.Context, rawURL string) (time.Time, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid URL: %v", err)
	}
	host := parsed.Hostname()
	port := parsed.Port()
	if port == "" {
		port = "443"
	}
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{},
		Config:    &tls.Config{ServerName: host},
	}
	conn, err := dialer.DialContext(ctx, "tcp", host+":"+port)
	if err != nil {
		return time.Time{}, fmt.Errorf("TLS dial failed: %v", err)
	}
	defer conn.Close()
	certs := conn.(*tls.Conn).ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return time.Time{}, fmt.Errorf("no certificates found")
	}
	return certs[0].NotAfter, nil
}

func checkHTTP(ctx context.Context, m *models.Monitor) (models.MonitorStatus, string) {
	method := m.Method
	if method == "" {
		method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, method, m.URL, strings.NewReader(m.Body))
	if err != nil {
		return models.StatusDown, fmt.Sprintf("request build error: %v", err)
	}

	// Parse and set custom headers
	if m.Headers != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(m.Headers), &headers); err == nil {
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return models.StatusDown, fmt.Sprintf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Build the set of accepted status codes.
	// AcceptedStatuses (comma-separated) takes priority; falls back to ExpectedStatus (default 200).
	accepted := parseAcceptedStatuses(m.AcceptedStatuses, m.ExpectedStatus)
	if !accepted[resp.StatusCode] {
		return models.StatusDown, fmt.Sprintf("unexpected status %d (accepted: %s)", resp.StatusCode, acceptedString(accepted))
	}

	// TLS certificate check — automatic for all HTTPS monitors
	if strings.HasPrefix(strings.ToLower(m.URL), "https://") {
		expiry, err := tlsCertExpiry(ctx, m.URL)
		if err != nil {
			// Don't fail the whole check on TLS dial error (HTTP already succeeded)
			return models.StatusUp, fmt.Sprintf("HTTP %d (TLS cert check failed: %v)", resp.StatusCode, err)
		}
		daysRemaining := int(time.Until(expiry).Hours() / 24)
		if daysRemaining <= 0 {
			// Cert is actually expired — mark DOWN
			return models.StatusDown, fmt.Sprintf("TLS certificate expired on %s", expiry.Format("2006-01-02"))
		}
		// Include expiry info in message; engine handles warning notifications
		return models.StatusUp, fmt.Sprintf("HTTP %d, TLS cert valid until %s (%d days)", resp.StatusCode, expiry.Format("2006-01-02"), daysRemaining)
	}

	return models.StatusUp, fmt.Sprintf("HTTP %d", resp.StatusCode)
}

// parseAcceptedStatuses builds a set of accepted HTTP status codes.
// If raw is non-empty (e.g. "200,202,204"), it is parsed; otherwise fallback is used (default 200).
func parseAcceptedStatuses(raw string, fallback int) map[int]bool {
	set := make(map[int]bool)
	raw = strings.TrimSpace(raw)
	if raw != "" {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if code, err := strconv.Atoi(part); err == nil && code > 0 {
				set[code] = true
			}
		}
	}
	if len(set) == 0 {
		if fallback == 0 {
			fallback = http.StatusOK
		}
		set[fallback] = true
	}
	return set
}

// acceptedString returns a human-readable sorted list of accepted status codes.
func acceptedString(accepted map[int]bool) string {
	codes := make([]int, 0, len(accepted))
	for c := range accepted {
		codes = append(codes, c)
	}
	// simple insertion sort (small slice)
	for i := 1; i < len(codes); i++ {
		for j := i; j > 0 && codes[j] < codes[j-1]; j-- {
			codes[j], codes[j-1] = codes[j-1], codes[j]
		}
	}
	parts := make([]string, len(codes))
	for i, c := range codes {
		parts[i] = strconv.Itoa(c)
	}
	return strings.Join(parts, ",")
}

// ─── Port ─────────────────────────────────────────────────────────────────────

func checkPort(ctx context.Context, m *models.Monitor) (models.MonitorStatus, string) {
	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return models.StatusDown, fmt.Sprintf("port unreachable: %v", err)
	}
	conn.Close()
	return models.StatusUp, fmt.Sprintf("port %d open", m.Port)
}

// ─── Ping ─────────────────────────────────────────────────────────────────────
// Note: ICMP ping requires raw sockets (root/CAP_NET_RAW). We use TCP echo
// as a fallback for environments without elevated privileges.

func checkPing(ctx context.Context, m *models.Monitor) (models.MonitorStatus, string) {
	// Try TCP connect to port 80 as a connectivity check
	// For real ICMP ping, use golang.org/x/net/icmp (requires privileges)
	addr := fmt.Sprintf("%s:80", m.Host)
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		// Try port 443
		addr = fmt.Sprintf("%s:443", m.Host)
		conn, err = d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return models.StatusDown, fmt.Sprintf("host unreachable: %v", err)
		}
	}
	conn.Close()
	return models.StatusUp, "host reachable"
}

// ─── Keyword ──────────────────────────────────────────────────────────────────

func checkKeyword(ctx context.Context, m *models.Monitor) (models.MonitorStatus, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL, nil)
	if err != nil {
		return models.StatusDown, fmt.Sprintf("request build error: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return models.StatusDown, fmt.Sprintf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return models.StatusDown, fmt.Sprintf("failed to read body: %v", err)
	}

	contains := strings.Contains(string(body), m.Keyword)

	switch m.KeywordMode {
	case "not_contains":
		if contains {
			return models.StatusDown, fmt.Sprintf("keyword %q found (should not be present)", m.Keyword)
		}
		return models.StatusUp, fmt.Sprintf("keyword %q not found (as expected)", m.Keyword)
	default: // "contains"
		if !contains {
			return models.StatusDown, fmt.Sprintf("keyword %q not found", m.Keyword)
		}
		return models.StatusUp, fmt.Sprintf("keyword %q found", m.Keyword)
	}
}

// ─── JSON API ─────────────────────────────────────────────────────────────────

func checkJSONAPI(ctx context.Context, m *models.Monitor) (models.MonitorStatus, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL, nil)
	if err != nil {
		return models.StatusDown, fmt.Sprintf("request build error: %v", err)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return models.StatusDown, fmt.Sprintf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var data interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return models.StatusDown, fmt.Sprintf("invalid JSON response: %v", err)
	}

	if m.JSONPath == "" {
		return models.StatusUp, "valid JSON response"
	}

	// Simple dot-notation path traversal (e.g. "status.code")
	value, err := jsonPathGet(data, m.JSONPath)
	if err != nil {
		return models.StatusDown, fmt.Sprintf("JSON path %q not found: %v", m.JSONPath, err)
	}

	if m.JSONExpected != "" {
		actual := fmt.Sprintf("%v", value)
		if actual != m.JSONExpected {
			return models.StatusDown, fmt.Sprintf("JSON path %q = %q, expected %q", m.JSONPath, actual, m.JSONExpected)
		}
	}

	return models.StatusUp, fmt.Sprintf("JSON path %q OK", m.JSONPath)
}

func jsonPathGet(data interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := data
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("not an object at %q", part)
		}
		val, exists := m[part]
		if !exists {
			return nil, fmt.Errorf("key %q not found", part)
		}
		current = val
	}
	return current, nil
}

// ─── UDP ──────────────────────────────────────────────────────────────────────

func checkUDP(ctx context.Context, m *models.Monitor) (models.MonitorStatus, string) {
	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return models.StatusDown, fmt.Sprintf("UDP dial failed: %v", err)
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline) //nolint:errcheck
	}

	payload := []byte(m.Body)
	if len(payload) == 0 {
		payload = []byte("\x00")
	}

	if _, err := conn.Write(payload); err != nil {
		return models.StatusDown, fmt.Sprintf("UDP write failed: %v", err)
	}

	// If no expected response, just sending is enough
	if m.JSONExpected == "" {
		return models.StatusUp, "UDP packet sent"
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return models.StatusDown, fmt.Sprintf("UDP read failed: %v", err)
	}

	response := string(buf[:n])
	if !strings.Contains(response, m.JSONExpected) {
		return models.StatusDown, fmt.Sprintf("unexpected UDP response: %q", response)
	}

	return models.StatusUp, "UDP response OK"
}

// ─── Response Time ────────────────────────────────────────────────────────────

func checkResponseTime(ctx context.Context, m *models.Monitor) (models.MonitorStatus, string) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL, nil)
	if err != nil {
		return models.StatusDown, fmt.Sprintf("request build error: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return models.StatusDown, fmt.Sprintf("request failed: %v", err)
	}
	resp.Body.Close()

	elapsed := time.Since(start).Milliseconds()
	if m.MaxResponseTimeMs > 0 && elapsed > int64(m.MaxResponseTimeMs) {
		return models.StatusDown, fmt.Sprintf("response time %dms exceeds threshold %dms", elapsed, m.MaxResponseTimeMs)
	}

	return models.StatusUp, fmt.Sprintf("response time %dms", elapsed)
}

// ─── DNS ──────────────────────────────────────────────────────────────────────

func checkDNS(ctx context.Context, m *models.Monitor) (models.MonitorStatus, string) {
	resolver := &net.Resolver{}

	switch strings.ToUpper(m.DNSRecordType) {
	case "A", "AAAA":
		addrs, err := resolver.LookupHost(ctx, m.DNSHost)
		if err != nil {
			return models.StatusDown, fmt.Sprintf("DNS lookup failed: %v", err)
		}
		if m.DNSExpected != "" {
			for _, addr := range addrs {
				if addr == m.DNSExpected {
					return models.StatusUp, fmt.Sprintf("DNS resolved to %s", addr)
				}
			}
			return models.StatusDown, fmt.Sprintf("expected %q not in results: %v", m.DNSExpected, addrs)
		}
		return models.StatusUp, fmt.Sprintf("resolved: %v", addrs)

	case "CNAME":
		cname, err := resolver.LookupCNAME(ctx, m.DNSHost)
		if err != nil {
			return models.StatusDown, fmt.Sprintf("CNAME lookup failed: %v", err)
		}
		if m.DNSExpected != "" && !strings.Contains(cname, m.DNSExpected) {
			return models.StatusDown, fmt.Sprintf("CNAME %q doesn't match expected %q", cname, m.DNSExpected)
		}
		return models.StatusUp, fmt.Sprintf("CNAME: %s", cname)

	case "MX":
		records, err := resolver.LookupMX(ctx, m.DNSHost)
		if err != nil {
			return models.StatusDown, fmt.Sprintf("MX lookup failed: %v", err)
		}
		return models.StatusUp, fmt.Sprintf("MX records: %d found", len(records))

	case "TXT":
		records, err := resolver.LookupTXT(ctx, m.DNSHost)
		if err != nil {
			return models.StatusDown, fmt.Sprintf("TXT lookup failed: %v", err)
		}
		if m.DNSExpected != "" {
			for _, r := range records {
				if strings.Contains(r, m.DNSExpected) {
					return models.StatusUp, "TXT record found"
				}
			}
			return models.StatusDown, fmt.Sprintf("expected TXT %q not found", m.DNSExpected)
		}
		return models.StatusUp, fmt.Sprintf("TXT records: %d found", len(records))

	default:
		return models.StatusDown, fmt.Sprintf("unsupported DNS record type: %s", m.DNSRecordType)
	}
}

// ─── SSL Certificate ──────────────────────────────────────────────────────────

func checkSSL(ctx context.Context, m *models.Monitor) (models.MonitorStatus, string) {
	domain := m.Domain
	addr := domain + ":443"

	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{},
		Config:    &tls.Config{ServerName: domain},
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return models.StatusDown, fmt.Sprintf("TLS connection failed: %v", err)
	}
	defer conn.Close()

	tlsConn := conn.(*tls.Conn)
	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return models.StatusDown, "no certificates found"
	}

	cert := certs[0]
	daysRemaining := int(time.Until(cert.NotAfter).Hours() / 24)
	warnDays := m.WarnDaysExpiry
	if warnDays == 0 {
		warnDays = 30
	}

	if daysRemaining <= 0 {
		return models.StatusDown, fmt.Sprintf("SSL certificate expired on %s", cert.NotAfter.Format("2006-01-02"))
	}
	if daysRemaining <= warnDays {
		return models.StatusDown, fmt.Sprintf("SSL certificate expires in %d days (%s)", daysRemaining, cert.NotAfter.Format("2006-01-02"))
	}

	return models.StatusUp, fmt.Sprintf("SSL valid, expires in %d days", daysRemaining)
}

// ─── Domain Expiry ────────────────────────────────────────────────────────────

func checkDomainExpiry(ctx context.Context, m *models.Monitor) (models.MonitorStatus, string) {
	// WHOIS lookup via TCP
	whoisServer := "whois.iana.org"
	domain := m.Domain

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", whoisServer+":43")
	if err != nil {
		return models.StatusDown, fmt.Sprintf("WHOIS connection failed: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "%s\r\n", domain)
	body, err := io.ReadAll(io.LimitReader(conn, 1<<16))
	if err != nil {
		return models.StatusDown, fmt.Sprintf("WHOIS read failed: %v", err)
	}

	// Parse expiry date from WHOIS response
	expiry, err := parseWhoisExpiry(string(body))
	if err != nil {
		return models.StatusDown, fmt.Sprintf("could not parse WHOIS expiry: %v", err)
	}

	daysRemaining := int(time.Until(expiry).Hours() / 24)
	warnDays := m.WarnDaysExpiry
	if warnDays == 0 {
		warnDays = 30
	}

	if daysRemaining <= 0 {
		return models.StatusDown, fmt.Sprintf("domain expired on %s", expiry.Format("2006-01-02"))
	}
	if daysRemaining <= warnDays {
		return models.StatusDown, fmt.Sprintf("domain expires in %d days (%s)", daysRemaining, expiry.Format("2006-01-02"))
	}

	return models.StatusUp, fmt.Sprintf("domain valid, expires in %d days", daysRemaining)
}

func parseWhoisExpiry(whois string) (time.Time, error) {
	// Common WHOIS date field names
	prefixes := []string{
		"Expiry Date:", "Expiration Date:", "Registry Expiry Date:",
		"paid-till:", "expires:", "Expires On:",
	}
	formats := []string{
		time.RFC3339, "2006-01-02", "2006-01-02T15:04:05Z",
		"02-Jan-2006", "2006-01-02 15:04:05",
	}

	for _, line := range strings.Split(whois, "\n") {
		for _, prefix := range prefixes {
			if strings.Contains(strings.ToLower(line), strings.ToLower(prefix)) {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) < 2 {
					continue
				}
				dateStr := strings.TrimSpace(parts[1])
				for _, format := range formats {
					if t, err := time.Parse(format, dateStr); err == nil {
						return t, nil
					}
				}
			}
		}
	}

	return time.Time{}, fmt.Errorf("expiry date not found in WHOIS response")
}
