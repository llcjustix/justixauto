// Command bootstrap-admin creates the first platform administrator of an empty
// database. By default it refuses once any active platform administrator exists.
// With -if-missing it instead succeeds without changes or reading credentials.
// The password is read from standard input, never from flags or the environment.
//
//	printf '%s' "$PASSWORD" | bootstrap-admin -login admin -email admin@example.com -name "Admin"
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"justixauto/internal/config"
	"justixauto/internal/modules/identity"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/database"
	"justixauto/internal/pkg/envx"
)

func main() {
	// Local development: .env fills in what the environment does not set.
	err := envx.LoadFile(".env")
	if err == nil {
		err = run()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bootstrap-admin:", err)
		var v *apperr.ValidationError
		if errors.As(err, &v) {
			for field, msg := range v.Fields {
				fmt.Fprintf(os.Stderr, "  %s: %s\n", field, msg)
			}
		}
		os.Exit(1)
	}
}

func run() error {
	login := flag.String("login", "", "sign-in login")
	email := flag.String("email", "", "email address")
	name := flag.String("name", "", "display name")
	ifMissing := flag.Bool("if-missing", false, "succeed without changes when a platform administrator already exists")
	flag.Parse()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	m, err := identity.New(db, identity.Config{Session: identity.DefaultSessionConfig})
	if err != nil {
		return err
	}
	ctx := context.Background()
	if *ifMissing {
		exists, err := m.Users.HasActivePlatformAdmin(ctx)
		if err != nil {
			return err
		}
		if exists {
			fmt.Println("platform administrator already exists; skipping bootstrap")
			return nil
		}
	}
	password, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if password == "" {
		return errors.New("password must be provided on standard input")
	}
	u, err := m.Users.Bootstrap(ctx, identity.BootstrapInput{
		DisplayName: *name, Login: *login, Email: *email, Password: strings.TrimRight(password, "\r\n"),
	})
	if err != nil {
		var appErr *apperr.Error
		if *ifMissing && errors.As(err, &appErr) && appErr.Code == "already_bootstrapped" {
			fmt.Println("platform administrator already exists; skipping bootstrap")
			return nil
		}
		return err
	}
	fmt.Printf("created platform administrator %s (%s)\n", *u.Login, u.ID)
	return nil
}
