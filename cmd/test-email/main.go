package main

import (
	"fmt"
	"log"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/email"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Check if email is configured
	if cfg.Email.Username == "" || cfg.Email.Password == "" {
		log.Fatal("Email credentials not configured. Please set EMAIL_USERNAME and EMAIL_PASSWORD in .env file")
	}

	fmt.Println("=== Email Service Test ===")
	fmt.Printf("SMTP Host: %s:%d\n", cfg.Email.Host, cfg.Email.Port)
	fmt.Printf("From: %s\n", cfg.Email.From)
	fmt.Printf("Frontend URL: %s\n", cfg.Frontend.URL)
	fmt.Println()

	// Create email service
	emailService := email.NewService(
		cfg.Email.Host,
		cfg.Email.Port,
		cfg.Email.Username,
		cfg.Email.Password,
		cfg.Email.From,
		cfg.Frontend.URL,
	)

	// Test recipient
	testEmail := "wns1826@naver.com"
	testUsername := "ultrathink"

	// Generate test tokens
	verificationToken := fmt.Sprintf("test-verification-%d", time.Now().Unix())
	resetToken := fmt.Sprintf("test-reset-%d", time.Now().Unix())

	// Send verification email
	fmt.Printf("📧 Sending verification email to %s...\n", testEmail)
	err = emailService.SendVerificationEmail(testEmail, testUsername, verificationToken)
	if err != nil {
		log.Printf("❌ Failed to send verification email: %v\n", err)
	} else {
		fmt.Println("✅ Verification email sent successfully!")
		fmt.Printf("   Token: %s\n", verificationToken)
		fmt.Printf("   URL: %s/verify-email?token=%s\n", cfg.Frontend.URL, verificationToken)
	}

	fmt.Println()

	// Wait a bit before sending next email
	time.Sleep(2 * time.Second)

	// Send password reset email
	fmt.Printf("📧 Sending password reset email to %s...\n", testEmail)
	err = emailService.SendPasswordResetEmail(testEmail, testUsername, resetToken)
	if err != nil {
		log.Printf("❌ Failed to send password reset email: %v\n", err)
	} else {
		fmt.Println("✅ Password reset email sent successfully!")
		fmt.Printf("   Token: %s\n", resetToken)
		fmt.Printf("   URL: %s/reset-password?token=%s\n", cfg.Frontend.URL, resetToken)
	}

	fmt.Println()
	fmt.Println("=== Test Complete ===")
	fmt.Println()
	fmt.Println("Please check your email inbox at:", testEmail)
	fmt.Println("Note: Check spam folder if you don't see the emails")
}
