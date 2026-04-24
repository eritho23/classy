package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"classy/internal/credentials"
	"classy/internal/generated/database"
	"classy/internal/handlers"
	"classy/internal/middleware"

	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5"
)

func getCSRFTrustedOrigins(origin *url.URL) []string {
	origins := map[string]struct{}{}

	add := func(candidate string) {
		if candidate == "" {
			return
		}
		origins[candidate] = struct{}{}
	}

	hostname := origin.Hostname()
	port := origin.Port()
	if port == "" {
		switch origin.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		}
	}
	add(origin.Host)
	add(hostname)
	if hostname != "" && port != "" {
		add(net.JoinHostPort(hostname, port))
	}
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		for _, alias := range []string{"localhost", "127.0.0.1", "::1"} {
			add(alias)
			if port != "" {
				add(net.JoinHostPort(alias, port))
			}
		}
	}

	trustedOrigins := make([]string, 0, len(origins))
	for allowedOrigin := range origins {
		trustedOrigins = append(trustedOrigins, allowedOrigin)
	}
	sort.Strings(trustedOrigins)

	return trustedOrigins
}

func getCSRFProtectionKey() []byte {
	csrfAuthKey, err := credentials.ReadCredential("csrf_auth_key")
	if err != nil {
		csrfAuthKey, _ = os.LookupEnv("CSRF_AUTH_KEY")
	}

	if csrfAuthKey != "" {
		hash := sha256.Sum256([]byte(csrfAuthKey))
		return hash[:]
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("failed to generate CSRF auth key: %v", err)
	}

	log.Print("CSRF auth key is not configured; using ephemeral key")
	return key
}

func normalizeNullOrigin(expectedOrigin *url.URL, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions && r.Method != http.MethodTrace {
			origin := r.Header.Get("Origin")
			secFetchSite := strings.TrimSpace(strings.ToLower(r.Header.Get("Sec-Fetch-Site")))
			if origin == "null" && (secFetchSite == "" || secFetchSite == "same-origin" || secFetchSite == "none") {
				hostMatchesExpected := r.Host == expectedOrigin.Host || r.Host == expectedOrigin.Hostname()
				if hostMatchesExpected {
					r.Header.Set("Origin", expectedOrigin.Scheme+"://"+expectedOrigin.Host)
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()

	databaseUrl, err := credentials.ReadCredential("database_url")
	if err != nil {
		var ok bool
		databaseUrl, ok = os.LookupEnv("DATABASE_URL")
		if !ok {
			log.Fatal("failed to lookup credentials dir and DATABASE_URL is not set")
		}
	}

	socketPath, exists := os.LookupEnv("HTTP_SOCKET_PATH")
	if !exists {
		log.Fatal("HTTP_SOCKET_PATH not set")
	}

	origin, exists := os.LookupEnv("ORIGIN")
	if !exists {
		log.Fatal("ORIGIN not set")
	}
	parsedOrigin, err := url.Parse(origin)
	if err != nil || parsedOrigin.Scheme == "" {
		log.Fatal("ORIGIN must be a valid absolute URL")
	}

	db, err := pgx.Connect(ctx, databaseUrl)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	q := queries.New(db)
	app := handlers.NewClassyApplication(q, db)

	_, err = os.Stat(socketPath)
	if err == nil {
		err = os.Remove(socketPath)
		if err != nil {
			log.Fatalf("failed to clear old socket: %v", err)
		}
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Failed to create listener: %v.\n", err)
	}

	mux := http.NewServeMux()
	app.RegisterRouteHandlers(mux)

	crossOriginProtection := http.NewCrossOriginProtection()
	err = crossOriginProtection.AddTrustedOrigin(origin)
	if err != nil {
		log.Fatalf("Failed to configure trusted origin: %v", err)
	}
	crossOriginProtection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Warn("cross-origin protection denied request")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("CSRF check failed"))
	}))

	muxWithMiddleware := middleware.CheckAuth(q, mux)
	trustedOrigins := getCSRFTrustedOrigins(parsedOrigin)
	slog.Info("configured csrf trusted origins", slog.Any("trusted_origins", trustedOrigins))
	csrfProtection := csrf.Protect(
		getCSRFProtectionKey(),
		csrf.Secure(parsedOrigin.Scheme == "https"),
		// Lax avoids false positives on legitimate top-level navigations.
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.TrustedOrigins(trustedOrigins),
		csrf.Path("/"),
		csrf.HttpOnly(true),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reason := "unknown"
			if failure := csrf.FailureReason(r); failure != nil {
				reason = failure.Error()
			}
			slog.Warn("csrf token validation failed",
				slog.String("reason", reason),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("host", r.Host),
				slog.String("origin", r.Header.Get("Origin")),
				slog.String("referer", r.Header.Get("Referer")),
				slog.String("sec_fetch_site", r.Header.Get("Sec-Fetch-Site")),
				slog.String("sec_fetch_mode", r.Header.Get("Sec-Fetch-Mode")),
				slog.String("sec_fetch_dest", r.Header.Get("Sec-Fetch-Dest")),
			)
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("CSRF token check failed"))
		})),
	)
	handler := normalizeNullOrigin(parsedOrigin, csrfProtection(muxWithMiddleware))
	server := &http.Server{
		Handler:           crossOriginProtection.Handler(handler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.Serve(listener); err != nil {
		log.Fatal(err)
	}

	if err := db.Close(ctx); err != nil {
		log.Printf("failed to close db: %v", err)
	}

	if err := listener.Close(); err != nil {
		log.Printf("failed to close listener: %v", err)
	}
}
