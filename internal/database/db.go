package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Options 数据库初始化选项
type Options struct {
	// Path SQLite 文件路径，如 ".data/data.db"
	Path string

	// BusyTimeout 数据库锁等待时间（默认 5s）
	BusyTimeout time.Duration

	// DisableForeignKeys 禁用外键约束（默认 false，推荐启用外键）
	DisableForeignKeys bool

	// MaxOpenConns 必须保持为 1，避免连接级 PRAGMA 在多个 SQLite 连接之间失效。
	MaxOpenConns int

	// MaxIdleConns 必须保持为 1，与 MaxOpenConns 一起固定为单连接模式。
	MaxIdleConns int
}

func (o Options) withDefaults() Options {
	if o.BusyTimeout == 0 {
		o.BusyTimeout = 5 * time.Second
	}
	if o.MaxOpenConns == 0 {
		o.MaxOpenConns = 1
	}
	if o.MaxIdleConns == 0 {
		o.MaxIdleConns = 1
	}
	return o
}

func (o Options) validate() error {
	switch {
	case o.BusyTimeout < 0:
		return errors.New("db: busy timeout must not be negative")
	case o.MaxOpenConns != 1:
		return fmt.Errorf("db: sqlite requires MaxOpenConns=1 to keep connection-level PRAGMAs reliable, got %d", o.MaxOpenConns)
	case o.MaxIdleConns != 1:
		return fmt.Errorf("db: sqlite requires MaxIdleConns=1 to keep connection-level PRAGMAs reliable, got %d", o.MaxIdleConns)
	default:
		return nil
	}
}

// DBContainer 数据库连接容器，管理生命周期
type DBContainer struct {
	db   *sql.DB
	path string
}

// Open 初始化并返回 DBContainer
func Open(ctx context.Context, opts Options) (*DBContainer, error) {
	opts = opts.withDefaults()
	ctx = normalizeContext(ctx)

	if err := opts.validate(); err != nil {
		return nil, err
	}

	if opts.Path == "" {
		return nil, errors.New("db: missing sqlite path")
	}

	// 确保数据目录存在
	if dir := filepath.Dir(opts.Path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("db: failed to create data dir %s: %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite", sqliteDSN(opts))
	if err != nil {
		return nil, fmt.Errorf("db: failed to open sqlite %s: %w", opts.Path, err)
	}

	// 连接池配置：SQLite 默认单连接最稳妥
	db.SetMaxOpenConns(opts.MaxOpenConns)
	db.SetMaxIdleConns(opts.MaxIdleConns)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if err := verifyPragmas(ctx, db, opts); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db: failed to ping sqlite %s: %w", opts.Path, err)
	}

	if err := RunMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &DBContainer{db: db, path: opts.Path}, nil
}

// DB 返回底层 *sql.DB
func (c *DBContainer) DB() *sql.DB {
	if c == nil {
		return nil
	}
	return c.db
}

// Path 返回数据库文件路径
func (c *DBContainer) Path() string {
	if c == nil {
		return ""
	}
	return c.path
}

// Close 关闭数据库连接
func (c *DBContainer) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	if err := c.db.Close(); err != nil {
		return fmt.Errorf("db: failed to close sqlite: %w", err)
	}
	return nil
}

func sqliteDSN(opts Options) string {
	params := url.Values{}
	if opts.BusyTimeout > 0 {
		params.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", opts.BusyTimeout.Milliseconds()))
	}
	if !opts.DisableForeignKeys {
		params.Add("_pragma", "foreign_keys(1)")
	}
	params.Add("_pragma", "journal_mode(WAL)")
	if encoded := params.Encode(); encoded != "" {
		return opts.Path + "?" + encoded
	}
	return opts.Path
}

func verifyPragmas(ctx context.Context, db *sql.DB, opts Options) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("db: failed to get sqlite conn: %w", err)
	}
	defer conn.Close()

	if !opts.DisableForeignKeys {
		var enabled int
		if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&enabled); err != nil {
			return fmt.Errorf("db: failed to verify foreign_keys: %w", err)
		}
		if enabled != 1 {
			return fmt.Errorf("db: expected foreign_keys pragma to be enabled, got %d", enabled)
		}
	}

	// 模板默认开启 WAL 模式
	var mode string
	if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
		return fmt.Errorf("db: failed to verify WAL: %w", err)
	}
	if !strings.EqualFold(mode, "wal") {
		return fmt.Errorf("db: expected WAL journal mode, got %q", mode)
	}

	if opts.BusyTimeout > 0 {
		var actualBusyTimeout int64
		if err := conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&actualBusyTimeout); err != nil {
			return fmt.Errorf("db: failed to verify busy_timeout: %w", err)
		}
		if actualBusyTimeout < opts.BusyTimeout.Milliseconds() {
			return fmt.Errorf("db: expected busy_timeout >= %dms, got %dms", opts.BusyTimeout.Milliseconds(), actualBusyTimeout)
		}
	}

	return nil
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
