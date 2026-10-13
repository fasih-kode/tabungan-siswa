package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/fasih/tabungan-siswa/internal/security"
)

const (
	databaseDSNEnv = "TABUNGAN_SISWA_DATABASE_DSN"
	expectedDBName = "tabungan_siswa_dev"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "bootstrap users:", err)
		os.Exit(1)
	}
}

func run() error {
	dsn := strings.TrimSpace(os.Getenv(databaseDSNEnv))
	if dsn == "" {
		return fmt.Errorf("%s is required; load the project's .env first", databaseDSNEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.OpenAndPing(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	var dbName string
	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&dbName); err != nil {
		return fmt.Errorf("identify target database: %w", err)
	}
	if dbName != expectedDBName {
		return fmt.Errorf("refusing to create accounts in database %q; expected %q", dbName, expectedDBName)
	}

	// Database identity is verified. Do not let interactive input consume the database timeout.
	cancel()

	fmt.Printf("Target database: %s\n", dbName)
	fmt.Printf("This command creates the initial ADMIN and WALI_KELAS accounts.\n")
	fmt.Printf("It does not assign the wali kelas to a class or academic year.\n\n")

	reader := bufio.NewReader(os.Stdin)
	adminUsername, err := readRequired(reader, "Username admin [admin]: ", "admin")
	if err != nil {
		return err
	}
	adminPassword, err := readPassword("Password admin (minimum 12 characters): ")
	if err != nil {
		return err
	}
	if len([]byte(adminPassword)) < 12 {
		return errors.New("admin password must contain at least 12 bytes")
	}
	adminConfirm, err := readPassword("Ulangi password admin: ")
	if err != nil {
		return err
	}
	if adminPassword != adminConfirm {
		return errors.New("admin password confirmation does not match")
	}

	waliUsername, err := readRequired(reader, "Username wali kelas [wali_kelas]: ", "wali_kelas")
	if err != nil {
		return err
	}
	if strings.EqualFold(adminUsername, waliUsername) {
		return errors.New("admin and wali kelas usernames must be different")
	}
	waliPassword, err := readPassword("Password wali kelas (minimum 12 characters): ")
	if err != nil {
		return err
	}
	if len([]byte(waliPassword)) < 12 {
		return errors.New("wali kelas password must contain at least 12 bytes")
	}
	waliConfirm, err := readPassword("Ulangi password wali kelas: ")
	if err != nil {
		return err
	}
	if waliPassword != waliConfirm {
		return errors.New("wali kelas password confirmation does not match")
	}

	fmt.Printf("\nKetik %q untuk mengonfirmasi pembuatan akun pada database ini: ", expectedDBName)
	confirmation, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read confirmation: %w", err)
	}
	if strings.TrimSpace(confirmation) != expectedDBName {
		return errors.New("confirmation did not match; no accounts were created")
	}

	// Start a fresh timeout after all interactive input is complete.
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	hasher, err := security.NewDefaultArgon2idHasher()
	if err != nil {
		return fmt.Errorf("initialize password hasher: %w", err)
	}
	adminHash, err := hasher.Hash(adminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	waliHash, err := hasher.Hash(waliPassword)
	if err != nil {
		return fmt.Errorf("hash wali kelas password: %w", err)
	}
	adminPassword = ""
	adminConfirm = ""
	waliPassword = ""
	waliConfirm = ""

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin bootstrap transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `LOCK TABLE users IN EXCLUSIVE MODE`); err != nil {
		return fmt.Errorf("lock users table for bootstrap: %w", err)
	}

	var adminExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE role = 'ADMIN')`).Scan(&adminExists); err != nil {
		return fmt.Errorf("check initial admin: %w", err)
	}
	if adminExists {
		return errors.New("an ADMIN account already exists; refusing to bootstrap accounts")
	}

	users := postgres.NewUserRepository(tx)
	for _, item := range []struct {
		username string
		password string
		role     domain.UserRole
	}{
		{adminUsername, adminHash, domain.RoleAdmin},
		{waliUsername, waliHash, domain.RoleWaliKelas},
	} {
		exists, err := users.ExistsByUsername(ctx, item.username)
		if err != nil {
			return fmt.Errorf("check username %q: %w", item.username, err)
		}
		if exists {
			return fmt.Errorf("username %q already exists; no accounts were created", item.username)
		}
		user, err := domain.NewUser(item.username, item.password, item.role)
		if err != nil {
			return fmt.Errorf("validate user %q: %w", item.username, err)
		}
		if err := users.Create(ctx, user); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return fmt.Errorf("username %q conflicts with an existing account", item.username)
			}
			return fmt.Errorf("create user %q: %w", item.username, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit bootstrap transaction: %w", err)
	}
	committed = true

	fmt.Println("Berhasil membuat akun ADMIN dan WALI_KELAS.")
	fmt.Println("Kata sandi disimpan sebagai hash Argon2id.")
	fmt.Println("Akun wali kelas belum ditugaskan ke kelas/tahun ajaran.")
	return nil
}

func readRequired(reader *bufio.Reader, prompt, fallback string) (string, error) {
	fmt.Print(prompt)
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read username: %w", err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	return value, nil
}

func readPassword(prompt string) (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", errors.New("password input requires an interactive terminal")
	}
	defer tty.Close()

	fmt.Fprint(tty, prompt)
	disableEcho := exec.Command("stty", "-echo")
	disableEcho.Stdin = tty
	if err := disableEcho.Run(); err != nil {
		return "", fmt.Errorf("disable terminal echo: %w", err)
	}
	defer func() {
		restoreEcho := exec.Command("stty", "echo")
		restoreEcho.Stdin = tty
		_ = restoreEcho.Run()
		fmt.Fprintln(tty)
	}()

	line := make([]byte, 0, 128)
	one := make([]byte, 1)
	for {
		n, readErr := tty.Read(one)
		if readErr != nil {
			return "", fmt.Errorf("read password: %w", readErr)
		}
		if n == 0 {
			continue
		}
		if one[0] == '\n' || one[0] == '\r' {
			break
		}
		if one[0] == 0x7f || one[0] == 0x08 {
			if len(line) > 0 {
				line = line[:len(line)-1]
			}
			continue
		}
		if one[0] == 0x03 {
			return "", errors.New("password input interrupted")
		}
		line = append(line, one[0])
		if len(line) > security.MaxPasswordBytes {
			return "", errors.New("password is too long")
		}
	}
	return string(line), nil
}
