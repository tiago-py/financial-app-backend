package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"financial-app-backend/internal/auth"
	"financial-app-backend/internal/cache"
	"financial-app-backend/internal/controller"
	"financial-app-backend/internal/model"
	"financial-app-backend/internal/repository"
	"financial-app-backend/internal/router"
	"financial-app-backend/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Use a disposable database only. The test creates an isolated schema and removes it.
func TestAccountCreationAndPasswordChange(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run database integration tests")
	}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 1
	schema := fmt.Sprintf("account_password_%d", time.Now().UnixNano())
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = pool.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE") }()
	migrations, err := filepath.Glob("../../migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range migrations {
		sql, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
	redis := miniredis.RunT(t)
	cached, err := cache.Open(ctx, redis.Addr(), "", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer cached.Close()
	store := repository.New(pool)
	svc := service.New(store, cached, auth.NewManager("integration-test-secret-at-least-32-characters", time.Hour))
	handler := router.New(controller.New(svc), svc, "http://localhost:3000", slog.New(slog.NewTextHandler(io.Discard, nil)))
	hash, _ := bcrypt.GenerateFromPassword([]byte("original-password"), bcrypt.MinCost)
	users := make([]model.User, 0, 3)
	for i, role := range []string{"user", "user", "admin"} {
		u, err := store.CreateUser(ctx, role, fmt.Sprintf("user%d@example.test", i), string(hash), role)
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, u)
	}
	request := func(method, path, token string, body any, status int) *httptest.ResponseRecorder {
		t.Helper()
		encoded, _ := json.Marshal(body)
		r := httptest.NewRequest(method, "/api/v1"+path, bytes.NewReader(encoded))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, w.Code, status, w.Body.String())
		}
		return w
	}
	tokens := make([]string, len(users))
	for i, u := range users {
		_, token, err := svc.Login(ctx, u.Email, "original-password")
		if err != nil {
			t.Fatal(err)
		}
		tokens[i] = token
	}
	accountInput := map[string]any{"name": "Minha conta", "institution": "Banco", "type": "checking", "openingBalanceCents": 125050, "openedOn": "2026-09-29", "color": "#0f6b57"}
	request("POST", "/accounts", "", accountInput, http.StatusUnauthorized)
	for _, i := range []int{0, 2} {
		result := request("POST", "/accounts", tokens[i], accountInput, http.StatusCreated)
		var account model.Account
		if err := json.Unmarshal(result.Body.Bytes(), &account); err != nil {
			t.Fatal(err)
		}
		if account.OpeningBalanceCents != 125050 {
			t.Fatal("incorrect opening balance")
		}
		request("GET", "/accounts/"+account.ID, tokens[i], nil, http.StatusOK)
		request("GET", "/accounts/"+account.ID, tokens[1], nil, http.StatusNotFound)
	}
	list := request("GET", "/accounts", tokens[1], nil, http.StatusOK)
	var empty struct {
		Items []model.Account `json:"items"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &empty); err != nil || len(empty.Items) != 0 {
		t.Fatal("cross-user accounts visible")
	}
	accountInput["ownerId"] = users[1].ID
	request("POST", "/accounts", tokens[0], accountInput, http.StatusBadRequest)
	delete(accountInput, "ownerId")
	accountInput["name"] = ""
	request("POST", "/accounts", tokens[0], accountInput, http.StatusUnprocessableEntity)
	request("GET", "/admin/users", tokens[0], nil, http.StatusForbidden)

	change := map[string]string{"currentPassword": "wrong-password", "newPassword": "new-password-123"}
	request("PATCH", "/me/password", "", change, http.StatusUnauthorized)
	request("PATCH", "/me/password", tokens[0], change, http.StatusUnprocessableEntity)
	request("GET", "/me", tokens[0], nil, http.StatusOK)
	change["currentPassword"] = "original-password"
	change["newPassword"] = "short"
	request("PATCH", "/me/password", tokens[0], change, http.StatusUnprocessableEntity)
	change["newPassword"] = "original-password"
	request("PATCH", "/me/password", tokens[0], change, http.StatusUnprocessableEntity)
	change["newPassword"] = "new-password-123"
	change["userId"] = users[1].ID
	request("PATCH", "/me/password", tokens[0], change, http.StatusBadRequest)
	delete(change, "userId")
	_, otherSession, _ := svc.Login(ctx, users[0].Email, "original-password")
	changed := request("PATCH", "/me/password", tokens[0], change, http.StatusNoContent)
	if !strings.Contains(changed.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatal("session cookie not cleared")
	}
	request("GET", "/me", tokens[0], nil, http.StatusUnauthorized)
	request("GET", "/me", otherSession, nil, http.StatusUnauthorized)
	request("GET", "/me", tokens[1], nil, http.StatusOK)
	request("POST", "/auth/login", "", map[string]string{"email": users[0].Email, "password": "original-password"}, http.StatusUnauthorized)
	request("POST", "/auth/login", "", map[string]string{"email": users[0].Email, "password": "new-password-123"}, http.StatusOK)
	savedHash, version, err := store.PasswordState(ctx, users[0].ID)
	if err != nil || version != 1 || savedHash == change["newPassword"] || bcrypt.CompareHashAndPassword([]byte(savedHash), []byte(change["newPassword"])) != nil {
		t.Fatal("password was not securely updated")
	}
	// A stale concurrent change cannot overwrite the successful change.
	if err := store.ChangePassword(ctx, users[0].ID, string(hash), string(hash)); err != model.ErrConflict {
		t.Fatalf("stale password update: %v", err)
	}
	// Changing the administrator password must survive bootstrap on restart.
	request("PATCH", "/me/password", tokens[2], change, http.StatusNoContent)
	if err := svc.BootstrapAdmin(ctx, "Administrator", users[2].Email, "original-password"); err != nil {
		t.Fatal(err)
	}
	request("POST", "/auth/login", "", map[string]string{"email": users[2].Email, "password": "new-password-123"}, http.StatusOK)
}
