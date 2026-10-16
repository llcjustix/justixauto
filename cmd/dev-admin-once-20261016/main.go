package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"justixauto/internal/modules/identity"
	"justixauto/internal/pkg/apperr"
)

const (
	legacySource = "3cc124c5a5858a5eaa2991edafc2771ee7232966"
	flatSource   = "96698c1ba6c4ef73cf73c17a8ff0464373e84840"
	// This credential exists only for the approved, temporary DEV operation.
	// #nosec G101 -- User-approved temporary DEV bootstrap credential; removed after one-time execution.
	adminPassword = "Z7!qN4@wK9#rT2$yV8%fH3&mL6^pC1*eS5"
)

func main() {
	expected := os.Getenv("OPERATION_SOURCE_SHA")
	if expected != legacySource && expected != flatSource {
		fail("source")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	db, err := open(ctx)
	if err != nil {
		fail("connect")
	}
	sqlDB, err := db.DB()
	if err != nil {
		fail("pool")
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Minute)

	mod, err := identity.New(db, identity.Config{Session: identity.DefaultSessionConfig})
	if err != nil {
		fail("identity")
	}
	_, err = mod.Users.Bootstrap(ctx, identity.BootstrapInput{
		DisplayName: "Platform Administrator",
		Login:       "admin",
		Email:       "admin@justixauto.test",
		Password:    adminPassword,
	})
	if err == nil {
		fmt.Println("outcome=created")
		return
	}
	var coded *apperr.Error
	if errors.As(err, &coded) && coded.Code == "already_bootstrapped" {
		fmt.Println("outcome=skip code=already_bootstrapped")
		return
	}
	if errors.As(err, &coded) {
		fail("bootstrap_" + coded.Code)
	}
	fail("bootstrap")
}

func open(ctx context.Context) (*gorm.DB, error) {
	for _, key := range []string{"PGHOST", "PGPORT", "PGDATABASE", "PGUSER", "PGPASSWORD"} {
		if os.Getenv(key) == "" {
			return nil, errors.New("database configuration missing")
		}
	}
	dsnURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(os.Getenv("PGUSER"), os.Getenv("PGPASSWORD")),
		Host:   os.Getenv("PGHOST") + ":" + os.Getenv("PGPORT"),
		Path:   os.Getenv("PGDATABASE"),
	}
	q := dsnURL.Query()
	q.Set("sslmode", "disable")
	q.Set("connect_timeout", "10")
	q.Set("statement_timeout", "15000")
	q.Set("lock_timeout", "5000")
	dsnURL.RawQuery = q.Encode()
	db, err := gorm.Open(postgres.Open(dsnURL.String()), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
		NowFunc:        func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, err
	}
	if err := db.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		return nil, err
	}
	return db, nil
}

func fail(stage string) {
	fmt.Printf("outcome=failed stage=%s\n", stage)
	os.Exit(1)
}
