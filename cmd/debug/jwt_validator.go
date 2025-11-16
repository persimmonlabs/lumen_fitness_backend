package main

import (
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

// This tool helps debug JWT token validation issues
// Usage: go run cmd/debug/jwt_test.go <token>

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/debug/jwt_test.go <jwt-token>")
		fmt.Println("\nThis tool validates a Supabase JWT token using the SUPABASE_JWT_SECRET")
		os.Exit(1)
	}

	tokenString := os.Args[1]
	secret := os.Getenv("SUPABASE_JWT_SECRET")

	if secret == "" {
		fmt.Println("Error: SUPABASE_JWT_SECRET environment variable not set")
		fmt.Println("\nSet it with:")
		fmt.Println("  export SUPABASE_JWT_SECRET='your-secret-here'")
		os.Exit(1)
	}

	fmt.Println("=== JWT Token Validation Test ===\n")
	fmt.Printf("Token length: %d characters\n", len(tokenString))
	fmt.Printf("Secret length: %d characters\n\n", len(secret))

	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		fmt.Printf("❌ Token validation FAILED: %v\n\n", err)

		// Try to parse without validation to see claims
		fmt.Println("Attempting to decode token without validation...")
		parser := jwt.NewParser(jwt.WithoutClaimsValidation())
		unvalidatedToken, parseErr := parser.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if parseErr == nil {
			if claims, ok := unvalidatedToken.Claims.(jwt.MapClaims); ok {
				fmt.Println("\n📋 Token Claims (unvalidated):")
				for key, value := range claims {
					fmt.Printf("  %s: %v\n", key, value)
				}
			}
		}

		os.Exit(1)
	}

	if !token.Valid {
		fmt.Println("❌ Token is INVALID (signature check failed)\n")
		os.Exit(1)
	}

	fmt.Println("✅ Token signature is VALID\n")

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		fmt.Println("❌ Failed to extract claims from token\n")
		os.Exit(1)
	}

	fmt.Println("📋 Token Claims:")
	for key, value := range claims {
		fmt.Printf("  %s: %v\n", key, value)
	}
	fmt.Println()

	// Check for required claims
	sub, hasSub := claims["sub"]
	exp, hasExp := claims["exp"]
	role, hasRole := claims["role"]

	fmt.Println("🔍 Required Claims Check:")

	if hasSub {
		fmt.Printf("  ✅ sub (user ID): %v\n", sub)
	} else {
		fmt.Println("  ❌ sub (user ID): MISSING")
	}

	if hasExp {
		fmt.Printf("  ✅ exp (expiration): %v\n", exp)
	} else {
		fmt.Println("  ❌ exp (expiration): MISSING")
	}

	if hasRole {
		fmt.Printf("  ℹ️  role: %v\n", role)
	} else {
		fmt.Println("  ⚠️  role: MISSING (not required but usually present)")
	}

	fmt.Println("\n=== Validation Summary ===")
	fmt.Println("✅ JWT token is valid and can be used for authentication")
}
