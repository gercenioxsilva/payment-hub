package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/you/clickpay-go-poc/internal/config"
)

func main() {
    cfg := config.Load()
    ctx := context.Background()

    db, err := pgxpool.New(ctx, cfg.DBURL)
    if err != nil { log.Fatal(err) }
    defer db.Close()

    dir := "internal/persistence/migrations"
    entries, err := os.ReadDir(dir)
    if err != nil { log.Fatal(err) }

    for _, e := range entries {
        if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") { continue }
        b, err := os.ReadFile(filepath.Join(dir, e.Name()))
        if err != nil { log.Fatal(err) }
        sql := string(b)
        _, err = db.Exec(ctx, sql)
        if err != nil { log.Fatalf("migration %s failed: %v", e.Name(), err) }
        fmt.Println("applied:", e.Name())
    }
}

