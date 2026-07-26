package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/juho05/sheetopia-sync/database"
	"github.com/juho05/sheetopia-sync/storage"
)

func usersList(queries *database.Queries) error {
	users, err := queries.FindUsers(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("Users (%d):\n", len(users))
	for _, u := range users {
		fmt.Println("  -", u.Name)
	}
	return nil
}

func usersCreate(args []string, queries *database.Queries) error {
	if len(args) < 4 {
		fmt.Println("USAGE:", args[0], "users create <name>")
		return ErrUsage
	}

	password, err := inputNewPassword("Enter password")
	if err != nil {
		return err
	}

	passwordHash, err := database.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	err = queries.CreateUser(context.Background(), database.CreateUserParams{
		Name:         args[3],
		PasswordHash: passwordHash,
	})
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			if sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY {
				return errors.New("user already exists")
			}
		}
		return err
	}
	fmt.Printf("Created user '%s'.\n", args[3])
	return nil
}

func usersDelete(args []string, queries *database.Queries) error {
	if len(args) < 4 {
		fmt.Println("USAGE:", args[0], "users delete <name>")
		return ErrUsage
	}
	result, err := queries.DeleteUser(context.Background(), args[3])
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	affectedRows, err := result.RowsAffected()
	if err == nil {
		if affectedRows == 0 {
			return fmt.Errorf("user '%s' does not exist", args[3])
		}
	}

	err = storage.DeleteUserScores(args[3])
	if err != nil {
		return fmt.Errorf("user '%s' was deleted from the database but its score files at %s remain: %w", args[3], storage.UserScoresDir(args[3]), err)
	}

	fmt.Printf("Deleted user '%s'.\n", args[3])
	return nil
}

func usersUpdate(args []string, db *sql.DB, queries *database.Queries) error {
	if len(args) < 5 {
		fmt.Println("USAGE:", args[0], "users update <name/password> <name>")
		return ErrUsage
	}
	switch args[3] {
	case "name":
		return usersChangeName(args[4], db, queries)
	case "password":
		return usersChangePassword(args[4], queries)
	default:
		fmt.Println("USAGE:", args[0], "users update <name/password> <name>")
		return ErrUsage
	}
}

func usersChangeName(user string, db *sql.DB, queries *database.Queries) error {
	name, err := input("Enter new name")
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("name must not be empty")
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	result, err := queries.WithTx(tx).UpdateUserName(ctx, database.UpdateUserNameParams{
		Name:   name,
		Name_2: user,
	})
	if err != nil {
		return fmt.Errorf("update user name in db: %w", err)
	}
	affectedRows, err := result.RowsAffected()
	if err == nil {
		if affectedRows == 0 {
			return fmt.Errorf("user '%s' does not exist", user)
		}
	}

	err = storage.RenameUserScores(user, name)
	if err != nil {
		return fmt.Errorf("rename score files: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	fmt.Printf("Changed name from '%s' to '%s'.\n", user, name)
	return nil
}

func usersChangePassword(user string, queries *database.Queries) error {
	password, err := inputNewPassword("Enter new password")
	if err != nil {
		return err
	}

	passwordHash, err := database.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	result, err := queries.UpdateUserPassword(context.Background(), database.UpdateUserPasswordParams{
		Name:         user,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return fmt.Errorf("update user password in db: %w", err)
	}
	affectedRows, err := result.RowsAffected()
	if err == nil {
		if affectedRows == 0 {
			return fmt.Errorf("user '%s' does not exist", user)
		}
	}

	fmt.Printf("Changed password of user '%s'.\n", user)
	return nil
}

func users(args []string, db *sql.DB, queries *database.Queries) error {
	if len(args) < 3 {
		fmt.Println("USAGE:", args[0], "users <command>\n\nCOMMANDS:\n  create\n  list\n  update\n  delete")
		return ErrUsage
	}
	var err error
	switch args[2] {
	case "list":
		err = usersList(queries)
	case "create":
		err = usersCreate(args, queries)
	case "update":
		err = usersUpdate(args, db, queries)
	case "delete":
		err = usersDelete(args, queries)
	default:
		fmt.Println("Unknown command")
		fmt.Println("USAGE:", args[0], "users <command>\n\nCOMMANDS:\n  list\n  create\n  update\n  delete")
		return ErrUsage
	}
	return err
}
