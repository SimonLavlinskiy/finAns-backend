package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var direction string
	flag.StringVar(&direction, "direction", "up", "migration direction: up or down")
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}

	m, err := migrate.New("file://db/migrations", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate init: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	switch direction {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	default:
		fmt.Fprintf(os.Stderr, "unknown direction: %s\n", direction)
		os.Exit(1)
	}

	if err != nil && err != migrate.ErrNoChange {
		fmt.Fprintf(os.Stderr, "migrate %s: %v\n", direction, err)
		os.Exit(1)
	}

	fmt.Printf("migrate %s: ok\n", direction)
}
