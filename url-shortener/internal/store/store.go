package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"url-shortener/internal/cache"
	"url-shortener/internal/util"
)

// ── Errors ────────────────────────────────────────────────────────────────────

var (
	ErrNotFound      = errors.New("not found")
	ErrExpired       = errors.New("link has expired")
	ErrAliasConflict = errors.New("alias already in use")
)

// ── Models ────────────────────────────────────────────────────────────────────

type URL struct {
	ID        int64
	Code      string
	Original  string
	Alias     *string
	ExpiresAt *time.Time
	CreatedAt time.Time
}

type Click struct {
	ID        int64
	URLID     int64
	IP        *string
	UserAgent *string
	Referer   *string
	CreatedAt time.Time
}

type ClickSummary struct {
	Total       int64      `json:"total"`
	LastClickAt *time.Time `json:"last_click_at,omitempty"`
}

// ── Store ─────────────────────────────────────────────────────────────────────

const (
	counterKey   = "counter:url"
	cachePrefix  = "url:"
	defaultCache = 24 * time.Hour
)

type Store struct {
	pool  *pgxpool.Pool
	cache *cache.Client
}

func New(pool *pgxpool.Pool, cache *cache.Client) *Store {
	return &Store{pool: pool, cache: cache}
}

// ── Shorten ───────────────────────────────────────────────────────────────────

type ShortenRequest struct {
	Original       string
	Alias          string
	TTLDays        int
	DefaultTTLDays int
}

type ShortenResponse struct {
	Code      string     `json:"code"`
	ShortURL  string     `json:"short_url"`
	Original  string     `json:"original"`
	Alias     *string    `json:"alias,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (s *Store) Shorten(ctx context.Context, baseURL string, req ShortenRequest) (*ShortenResponse, error) {
	url := &URL{Original: req.Original}

	// Resolve TTL
	ttlDays := req.TTLDays
	if ttlDays == 0 {
		ttlDays = req.DefaultTTLDays
	}
	if ttlDays > 0 {
		t := time.Now().UTC().AddDate(0, 0, ttlDays)
		url.ExpiresAt = &t
	}

	// Check alias availability
	if req.Alias != "" {
		existing, err := s.getByAlias(ctx, req.Alias)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("check alias: %w", err)
		}
		if existing != nil {
			return nil, ErrAliasConflict
		}
		url.Alias = &req.Alias
	}

	// Atomic counter → Base62 code
	seq, err := s.cache.Increment(ctx, counterKey)
	if err != nil {
		return nil, fmt.Errorf("increment counter: %w", err)
	}
	url.Code = util.Encode(seq)

	// Persist to Postgres
	created, err := s.createURL(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("create url: %w", err)
	}

	// Prime cache
	s.setCache(ctx, created)

	code := created.Code
	if created.Alias != nil && *created.Alias != "" {
		code = *created.Alias
	}

	return &ShortenResponse{
		Code:      code,
		ShortURL:  baseURL + "/" + code,
		Original:  created.Original,
		Alias:     created.Alias,
		ExpiresAt: created.ExpiresAt,
		CreatedAt: created.CreatedAt,
	}, nil
}

// ── Resolve ───────────────────────────────────────────────────────────────────

func (s *Store) Resolve(ctx context.Context, code string) (*URL, error) {
	// Cache hit
	if val, err := s.cache.Get(ctx, cachePrefix+code); err == nil && val != "" {
		return &URL{Code: code, Original: val}, nil
	}

	// Postgres — try code then alias
	url, err := s.getByCode(ctx, code)
	if errors.Is(err, ErrNotFound) {
		url, err = s.getByAlias(ctx, code)
	}
	if errors.Is(err, ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Expiry check
	if url.ExpiresAt != nil && time.Now().After(*url.ExpiresAt) {
		return nil, ErrExpired
	}

	s.setCache(ctx, url)
	return url, nil
}

// ── RecordClick ───────────────────────────────────────────────────────────────

// RecordClick is fire-and-forget — always call it in a goroutine.
func (s *Store) RecordClick(urlID int64, ip, userAgent, referer string) {
	const q = `INSERT INTO clicks (url_id, ip, user_agent, referer) VALUES ($1, $2, $3, $4)`
	_, _ = s.pool.Exec(context.Background(), q,
		urlID, strPtr(ip), strPtr(userAgent), strPtr(referer),
	)
}

// ── Stats ─────────────────────────────────────────────────────────────────────

func (s *Store) GetStats(ctx context.Context, code string) (*URL, *ClickSummary, error) {
	url, err := s.getByCode(ctx, code)
	if errors.Is(err, ErrNotFound) {
		url, err = s.getByAlias(ctx, code)
	}
	if errors.Is(err, ErrNotFound) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	const q = `SELECT COUNT(*), MAX(created_at) FROM clicks WHERE url_id = $1`
	var summary ClickSummary
	if err := s.pool.QueryRow(ctx, q, url.ID).Scan(&summary.Total, &summary.LastClickAt); err != nil {
		return nil, nil, fmt.Errorf("get stats: %w", err)
	}

	return url, &summary, nil
}

// ── Private helpers ───────────────────────────────────────────────────────────

func (s *Store) createURL(ctx context.Context, url *URL) (*URL, error) {
	const q = `
		INSERT INTO urls (code, original, alias, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, code, original, alias, expires_at, created_at`

	var out URL
	err := s.pool.QueryRow(ctx, q,
		url.Code, url.Original, url.Alias, url.ExpiresAt,
	).Scan(&out.ID, &out.Code, &out.Original, &out.Alias, &out.ExpiresAt, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) getByCode(ctx context.Context, code string) (*URL, error) {
	const q = `SELECT id, code, original, alias, expires_at, created_at FROM urls WHERE code = $1`
	return s.scanURL(s.pool.QueryRow(ctx, q, code))
}

func (s *Store) getByAlias(ctx context.Context, alias string) (*URL, error) {
	const q = `SELECT id, code, original, alias, expires_at, created_at FROM urls WHERE alias = $1`
	return s.scanURL(s.pool.QueryRow(ctx, q, alias))
}

func (s *Store) scanURL(row pgx.Row) (*URL, error) {
	var url URL
	err := row.Scan(&url.ID, &url.Code, &url.Original, &url.Alias, &url.ExpiresAt, &url.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &url, nil
}

func (s *Store) setCache(ctx context.Context, url *URL) {
	ttl := defaultCache
	if url.ExpiresAt != nil {
		if remaining := time.Until(*url.ExpiresAt); remaining > 0 && remaining < ttl {
			ttl = remaining
		}
	}
	_ = s.cache.Set(ctx, cachePrefix+url.Code, url.Original, ttl)
	if url.Alias != nil && *url.Alias != "" {
		_ = s.cache.Set(ctx, cachePrefix+*url.Alias, url.Original, ttl)
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
