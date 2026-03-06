package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

var tables = map[string]bool{
	"tp_activity":   true,
	"tp_banner":     true,
	"tp_category":   true,
	"tp_chapter":    true,
	"tp_comment":    true,
	"tp_community":  true,
	"tp_contact":    true,
	"tp_course":     true,
	"tp_material":   true,
	"tp_news":       true,
	"tp_notice":     true,
	"tp_noticetype": true,
	"tp_user":       true,
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS tp_activity (
  id INTEGER PRIMARY KEY,
  category TEXT,
  title TEXT,
  picPath TEXT,
  startDate TEXT,
  endDate TEXT,
  sponsor TEXT,
  content TEXT,
  position TEXT,
  signUpNum INTEGER,
  maxNum INTEGER,
  signUpEndDate TEXT,
  isTop TEXT,
  status TEXT
);
CREATE TABLE IF NOT EXISTS tp_banner (
  id INTEGER PRIMARY KEY,
  advImg TEXT,
  advTitle TEXT,
  type TEXT,
  status TEXT
);
CREATE TABLE IF NOT EXISTS tp_category (
  id INTEGER PRIMARY KEY,
  categoryName TEXT,
  appType TEXT,
  status TEXT
);
CREATE TABLE IF NOT EXISTS tp_chapter (
  id INTEGER PRIMARY KEY,
  uId TEXT,
  cId INTEGER,
  name TEXT,
  watch TEXT,
  status TEXT
);
CREATE TABLE IF NOT EXISTS tp_comment (
  id INTEGER PRIMARY KEY,
  content TEXT,
  commentDate TEXT,
  newsId INTEGER,
  userName TEXT,
  likeNum INTEGER,
  status TEXT
);
CREATE TABLE IF NOT EXISTS tp_community (
  id INTEGER PRIMARY KEY,
  name TEXT,
  sort TEXT,
  create_time TEXT,
  status TEXT
);
CREATE TABLE IF NOT EXISTS tp_contact (
  id INTEGER PRIMARY KEY,
  uId INTEGER,
  relationship TEXT,
  telephone TEXT,
  alternatePhone TEXT,
  createTime TEXT,
  updateTime TEXT,
  status TEXT
);
CREATE TABLE IF NOT EXISTS tp_course (
  id INTEGER PRIMARY KEY,
  title TEXT,
  content TEXT,
  cover TEXT,
  level TEXT,
  video TEXT,
  duration TEXT,
  collection TEXT,
  progress TEXT,
  status TEXT
);
CREATE TABLE IF NOT EXISTS tp_material (
  id INTEGER PRIMARY KEY,
  materialName TEXT,
  fileName TEXT,
  url TEXT,
  createTime TEXT,
  status TEXT
);
CREATE TABLE IF NOT EXISTS tp_news (
  id INTEGER PRIMARY KEY,
  title TEXT,
  subTitle TEXT,
  content TEXT,
  cover TEXT,
  publishDate TEXT,
  tags TEXT,
  hot TEXT,
  commentNum INTEGER,
  likeNum INTEGER,
  readNum INTEGER,
  updateTime TEXT,
  createTime TEXT,
  remark TEXT,
  appType TEXT,
  top TEXT,
  categoryId INTEGER,
  status TEXT,
  createBy TEXT
);
CREATE TABLE IF NOT EXISTS tp_notice (
  id INTEGER PRIMARY KEY,
  noticeTitle TEXT,
  noticeStatus INTEGER,
  contentNotice TEXT,
  releaseUnit TEXT,
  phone TEXT,
  createTime TEXT,
  expressId INTEGER,
  status TEXT
);
CREATE TABLE IF NOT EXISTS tp_noticetype (
  id INTEGER PRIMARY KEY,
  noticeName TEXT
);
CREATE TABLE IF NOT EXISTS tp_user (
  id INTEGER PRIMARY KEY,
  userName TEXT,
  passWord TEXT,
  avatar TEXT,
  nickName TEXT,
  phonenumber TEXT,
  SMSCode TEXT,
  sex TEXT,
  email TEXT,
  idCard TEXT,
  points TEXT,
  money REAL,
  address TEXT,
  introduction TEXT,
  createTime TEXT,
  updateTime TEXT,
  status TEXT
);
`

func main() {
	source := env("SOURCE_SQL", filepath.Clean("../health/health.sql"))
	target := env("SQLITE_PATH", "health.db")

	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		panic(err)
	}

	db, err := sql.Open("sqlite", target)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if _, err = db.Exec(schemaSQL); err != nil {
		panic(err)
	}

	f, err := os.Open(source)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 20*1024*1024)

	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "INSERT INTO `") {
			continue
		}
		table := parseTable(line)
		if !tables[table] {
			continue
		}
		if _, err = tx.Exec(line); err != nil {
			tx.Rollback()
			panic(fmt.Errorf("import failed at %s: %w", table, err))
		}
		count++
	}
	if err = scanner.Err(); err != nil {
		tx.Rollback()
		panic(err)
	}
	if err = tx.Commit(); err != nil {
		panic(err)
	}

	fmt.Printf("SQLite database created: %s\n", target)
	fmt.Printf("Imported INSERT statements: %d\n", count)
}

func parseTable(line string) string {
	rest := strings.TrimPrefix(line, "INSERT INTO `")
	i := strings.Index(rest, "`")
	if i < 0 {
		return ""
	}
	return rest[:i]
}

func env(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
