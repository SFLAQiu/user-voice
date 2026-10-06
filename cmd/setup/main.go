// Package main 交互式配置生成引导：make setup。
package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/feedback/internal/config"
	"github.com/feedback/internal/db"
	"github.com/feedback/internal/setup"
)

func main() {
	out := flag.String("out", setup.ConfigPath(), "生成的配置文件路径")
	force := flag.Bool("force", false, "目标文件已存在时覆盖")
	migrate := flag.Bool("migrate", true, "生成后执行迁移验证连通性")
	flag.Parse()

	if _, err := os.Stat(*out); err == nil && !*force {
		fmt.Fprintf(os.Stderr, "错误：%s 已存在，如需覆盖请加 -force\n", *out)
		os.Exit(1)
	}

	jwtSecret := randHex(32)
	aesKey := randBase64(32)
	fmt.Println("已生成 JWT secret 与 AES key（随机）")

	dsn := promptDSN()
	adminUser, adminPass := promptAdmin()

	cfg := fmt.Sprintf(`server:
  addr: ":8080"
  read_timeout: 30s
  write_timeout: 30s

database:
  dsn: "%s"
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: 1h

jwt:
  secret: "%s"
  expire: 8h

encryption:
  # 32-byte base64 (openssl rand -base64 32)
  key: "%s"

admin:
  username: "%s"
  password: "%s"

login_limit:
  max_attempts: 5
  window: 5m

log:
  level: "info"
  file: ""

cors:
  allow_origins:
    - "http://localhost:5173"
`, dsn, jwtSecret, aesKey, adminUser, adminPass)

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "错误：", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, []byte(cfg), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "错误：", err)
		os.Exit(1)
	}
	fmt.Printf("配置已写入 %s\n", *out)
	fmt.Printf("管理员账号：%s\n管理员密码（仅显示一次，请妥善保存）：%s\n", adminUser, adminPass)

	if *migrate {
		if err := verifyAndMigrate(dsn); err != nil {
			fmt.Fprintf(os.Stderr, "迁移验证失败：%v\n请检查数据库地址、账号密码、库名是否正确，且 MySQL 可达。\n", err)
			os.Exit(1)
		}
		fmt.Println("数据库迁移验证通过")
	}
}

// promptDSN 交互式收集数据库连接信息并拼接标准 DSN。
func promptDSN() string {
	host := ask("数据库地址", "127.0.0.1")
	port := ask("端口", "3306")
	user := ask("用户名", "root")
	pass := askHidden("密码", "")
	name := ask("数据库名", "feedback")
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, pass, host, port, name)
}

// promptAdmin 收集管理员账号，密码默认随机 16 位。
func promptAdmin() (string, string) {
	user := ask("管理员用户名", "admin")
	pass := askHidden("管理员密码（回车随机生成）", "")
	if pass == "" {
		pass = randHex(8)
	}
	return user, pass
}

func verifyAndMigrate(dsn string) error {
	// 复用内部包；setup 以本仓库模块身份运行（go run ./cmd/setup）
	cfg := config.DatabaseConfig{
		DSN:             dsn,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	}
	gdb, err := db.Open(cfg)
	if err != nil {
		return err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func ask(prompt, def string) string {
	fmt.Printf("%s [%s]: ", prompt, def)
	return scanLine(def)
}

func askHidden(prompt, def string) string {
	fmt.Printf("%s: ", prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil || len(b) == 0 {
			return def
		}
		return strings.TrimSpace(string(b))
	}
	// 非终端（管道等）按普通行读取
	return scanLine(def)
}

var stdin = bufio.NewScanner(os.Stdin)

func scanLine(def string) string {
	if !stdin.Scan() {
		return def
	}
	s := strings.TrimSpace(stdin.Text())
	if s == "" {
		return def
	}
	return s
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func randBase64(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}
