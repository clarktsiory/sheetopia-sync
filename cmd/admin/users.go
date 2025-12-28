package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/juho05/sheetopia-sync/database"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
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

	var password string
	for password == "" {
		p1 := inputPassword("Enter password")
		p2 := inputPassword("Repeat password")
		if p1 == p2 {
			password = p1
		} else {
			fmt.Println("Passwords don't match. Try again.")
		}
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
	fmt.Printf("Deleted user '%s'.\n", args[3])
	return nil
}

func usersUpdate(args []string, queries *database.Queries) error {
	if len(args) < 5 {
		fmt.Println("USAGE:", args[0], "users update <name/password> <name>")
		return ErrUsage
	}
	switch args[3] {
	case "name":
		return usersChangeName(args[4], queries)
	case "password":
		return usersChangePassword(args[4], queries)
	default:
		fmt.Println("USAGE:", args[0], "users update <name/password> <name>")
		return ErrUsage
	}
}

func usersChangeName(user string, queries *database.Queries) error {
	name := input("Enter new name")

	result, err := queries.UpdateUserName(context.Background(), database.UpdateUserNameParams{
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

	fmt.Printf("Changed name from '%s' to '%s'.\n", user, name)
	return nil
}

func usersChangePassword(user string, queries *database.Queries) error {
	var password string
	for password == "" {
		p1 := inputPassword("Enter new password")
		p2 := inputPassword("Repeat password")
		if p1 == p2 {
			password = p1
		} else {
			fmt.Println("Passwords don't match. Try again.")
		}
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

func users(args []string, queries *database.Queries) error {
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
		err = usersUpdate(args, queries)
	case "delete":
		err = usersDelete(args, queries)
	default:
		fmt.Println("Unknown command")
		fmt.Println("USAGE:", args[0], "users <command>\n\nCOMMANDS:\n  list\n  create\n  update\n  delete")
		return ErrUsage
	}
	return err
}
