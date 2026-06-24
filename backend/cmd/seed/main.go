// Seed populates the database with minimal test data for manual verification.
//
// Users:
//   admin       admin@indieforge.test    admin123456  (role=admin)
//   devauthor   dev@indieforge.test      dev123456    (role=user, is_developer=true)
//   player1     player@indieforge.test   player123    (role=user)
//
// Games (created by devauthor):
//   pixel-knights — free, browser-playable placeholder
//   galaxy-quest  — paid (599 ₽), subscription enabled (299 ₽/mo, 20 % friend-pack discount)
//
// player1 owns both games; player1 has an active subscription to galaxy-quest.
//
// Usage: DATABASE_URL=postgres://... go run ./cmd/seed
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://indieforge:indieforge@localhost:5432/indieforge?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err := seed(ctx, pool); err != nil {
		log.Fatalf("seed: %v", err)
	}
	fmt.Println("Seed complete.")
	fmt.Println()
	fmt.Println("  admin       / admin123456  — admin panel")
	fmt.Println("  devauthor   / dev123456    — developer dashboard")
	fmt.Println("  player1     / player123    — library with 2 games + subscription")
}

func hashpw(password string) string {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func seed(ctx context.Context, pool *pgxpool.Pool) error {
	// --- users ---
	type user struct {
		id, username, email, password, role string
		isDev                               bool
	}
	users := []user{
		{"usr_seed01", "admin", "admin@indieforge.test", "admin123456", "admin", false},
		{"usr_seed02", "devauthor", "dev@indieforge.test", "dev123456", "user", true},
		{"usr_seed03", "player1", "player@indieforge.test", "player123", "user", false},
	}
	for _, u := range users {
		if _, err := pool.Exec(ctx, `
			INSERT INTO users (id, username, email, password_hash, role, is_developer)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO NOTHING`,
			u.id, u.username, u.email, hashpw(u.password), u.role, u.isDev,
		); err != nil {
			return fmt.Errorf("user %s: %w", u.username, err)
		}
		fmt.Printf("  user %s … ok\n", u.username)
	}

	// --- games ---
	if _, err := pool.Exec(ctx, `
		INSERT INTO games (
			id, slug, title, tagline, description, genre, tags,
			developer_id, developer_name,
			pricing_model, price,
			has_browser_build, browser_build_url,
			status
		) VALUES (
			'game_seed01', 'pixel-knights',
			'Pixel Knights', 'A retro pixel-art RPG adventure',
			'Explore hand-crafted dungeons, collect loot, and slay monsters in this lovingly-made pixel-art RPG.',
			'RPG', ARRAY['pixel-art','rpg','adventure'],
			'usr_seed02', 'devauthor',
			'free', 0,
			false, '',
			'published'
		) ON CONFLICT (id) DO NOTHING`,
	); err != nil {
		return fmt.Errorf("game pixel-knights: %w", err)
	}
	fmt.Println("  game pixel-knights … ok")

	if _, err := pool.Exec(ctx, `
		INSERT INTO games (
			id, slug, title, tagline, description, genre, tags,
			developer_id, developer_name,
			pricing_model, price, friend_pack_discount,
			sub_enabled, sub_price, sub_benefits, sub_chat_link,
			status
		) VALUES (
			'game_seed02', 'galaxy-quest',
			'Galaxy Quest', 'Space exploration meets roguelike action',
			'Build your ship, chart star systems, survive the void. Every run is different.',
			'Action', ARRAY['roguelike','space','action'],
			'usr_seed02', 'devauthor',
			'paid', 599, 20,
			true, 299,
			ARRAY['Early access to every update', 'Subscriber-only Discord channel'],
			'https://discord.gg/example',
			'published'
		) ON CONFLICT (id) DO NOTHING`,
	); err != nil {
		return fmt.Errorf("game galaxy-quest: %w", err)
	}
	fmt.Println("  game galaxy-quest … ok")

	// --- payments (simulated succeeded) ---
	type payment struct {
		id, userID, gameID, kind string
		amount                   int
	}
	payments := []payment{
		{"pay_seed01", "usr_seed03", "game_seed01", "free", 0},
		{"pay_seed02", "usr_seed03", "game_seed02", "purchase", 599},
		{"pay_seed03", "usr_seed03", "game_seed02", "subscription", 299},
	}
	for _, p := range payments {
		if _, err := pool.Exec(ctx, `
			INSERT INTO payments (id, user_id, game_id, kind, amount, commission_percent, commission_amount, status)
			VALUES ($1, $2, $3, $4, $5, 10, $6, 'succeeded')
			ON CONFLICT (id) DO NOTHING`,
			p.id, p.userID, p.gameID, p.kind, p.amount, p.amount/10,
		); err != nil {
			return fmt.Errorf("payment %s: %w", p.id, err)
		}
	}
	fmt.Println("  payments … ok")

	// --- ownerships ---
	type ownership struct {
		id, userID, gameID, otype string
		price                     int
	}
	ownerships := []ownership{
		{"own_seed01", "usr_seed03", "game_seed01", "free", 0},
		{"own_seed02", "usr_seed03", "game_seed02", "purchase", 599},
	}
	for _, o := range ownerships {
		if _, err := pool.Exec(ctx, `
			INSERT INTO ownerships (id, user_id, game_id, type, price)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (user_id, game_id) DO NOTHING`,
			o.id, o.userID, o.gameID, o.otype, o.price,
		); err != nil {
			return fmt.Errorf("ownership %s: %w", o.id, err)
		}
	}
	fmt.Println("  ownerships … ok")

	// --- subscription ---
	if _, err := pool.Exec(ctx, `
		INSERT INTO subscriptions (id, user_id, game_id, developer_id, price, active)
		VALUES ('sub_seed01', 'usr_seed03', 'game_seed02', 'usr_seed02', 299, true)
		ON CONFLICT (id) DO NOTHING`,
	); err != nil {
		return fmt.Errorf("subscription: %w", err)
	}
	fmt.Println("  subscription … ok")

	// --- game events for trending ---
	events := []struct{ id, gameID, etype string }{
		{"evt_s001", "game_seed01", "view"},
		{"evt_s002", "game_seed01", "play"},
		{"evt_s003", "game_seed01", "acquire"},
		{"evt_s004", "game_seed01", "view"},
		{"evt_s005", "game_seed02", "view"},
		{"evt_s006", "game_seed02", "acquire"},
		{"evt_s007", "game_seed02", "view"},
	}
	for _, ev := range events {
		if _, err := pool.Exec(ctx, `
			INSERT INTO game_events (id, game_id, type)
			VALUES ($1, $2, $3)
			ON CONFLICT (id) DO NOTHING`,
			ev.id, ev.gameID, ev.etype,
		); err != nil {
			return fmt.Errorf("event %s: %w", ev.id, err)
		}
	}
	fmt.Println("  game events … ok")

	// --- settings: enable home sections ---
	if _, err := pool.Exec(ctx, `
		UPDATE settings SET trending_enabled = true, popular_enabled = true, updated_at = now()
		WHERE id = true`,
	); err != nil {
		return fmt.Errorf("settings: %w", err)
	}
	fmt.Println("  settings (trending+popular enabled) … ok")

	return nil
}
