package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	// MaxOpenConns SQLite 最佳实践通常是 1（默认 1）
	MaxOpenConns int

	// MaxIdleConns 默认 1
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

// DBContainer 数据库连接容器，管理生命周期
type DBContainer struct {
	db   *sql.DB
	path string
}

// Open 初始化并返回 DBContainer
func Open(ctx context.Context, opts Options) (*DBContainer, error) {
	opts = opts.withDefaults()
	ctx = normalizeContext(ctx)

	if opts.Path == "" {
		return nil, errors.New("db: missing sqlite path")
	}

	// 确保数据目录存在
	if dir := filepath.Dir(opts.Path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("db: failed to create data dir %s: %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite", opts.Path)
	if err != nil {
		return nil, fmt.Errorf("db: failed to open sqlite %s: %w", opts.Path, err)
	}

	// 连接池配置：SQLite 默认单连接最稳妥
	db.SetMaxOpenConns(opts.MaxOpenConns)
	db.SetMaxIdleConns(opts.MaxIdleConns)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if err := applyPragmas(ctx, db, opts); err != nil {
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

func applyPragmas(ctx context.Context, db *sql.DB, opts Options) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("db: failed to get sqlite conn: %w", err)
	}
	defer conn.Close()

	// 避免 "database is locked"
	if opts.BusyTimeout > 0 {
		ms := opts.BusyTimeout.Milliseconds()
		// PRAGMA busy_timeout 不能可靠地使用占位符，使用常量拼接是安全且可控的。
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("PRAGMA busy_timeout = %d", ms)); err != nil { // PRAGMA 不支持占位符，内部拼接安全
			return fmt.Errorf("db: failed to set busy_timeout: %w", err)
		}
	}

	if !opts.DisableForeignKeys {
		if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
			return fmt.Errorf("db: failed to enable foreign_keys: %w", err)
		}
	}

	// 模板默认开启 WAL 模式
	var mode string
	if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode=WAL").Scan(&mode); err != nil {
		return fmt.Errorf("db: failed to enable WAL: %w", err)
	}

	return nil
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
