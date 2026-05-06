// Package orm provides the Bun ORM singleton wired to the application's
// shared *sql.DB. Call Configure() once after database.Connect has succeeded,
// then use Database() anywhere a *bun.DB is needed.
//
// This package intentionally exposes no domain models — add struct types in
// sibling files as the application's schema grows.
package orm
