// Password hashing utility for Simple Reminder
// Usage: go run tools/hash-password.go your-password-here

package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost is the default cost factor for bcrypt hashing.
	DefaultCost = 12
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <password>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s mySecurePassword123\n", os.Args[0])
		os.Exit(1)
	}

	password := os.Args[1]
	hash, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Bcrypt hash for password: %s\n", string(hash))
	fmt.Println("\nAdd this hash to your config.yml:")
	fmt.Printf("web_server_password: \"%s\"\n", string(hash))
	fmt.Println("\nOr set as environment variable:")
	fmt.Printf("export WEB_SERVER_PASSWORD=\"%s\"\n", string(hash))
}
