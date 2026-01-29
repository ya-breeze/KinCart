package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"kincart/internal/database"
	"kincart/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("expected 'add-family', 'add-user', or 'seed-categories' subcommands")
		os.Exit(1)
	}

	database.InitDB()

	switch os.Args[1] {
	case "add-family":
		addFamilyCmd := flag.NewFlagSet("add-family", flag.ExitOnError)
		familyName := addFamilyCmd.String("name", "", "Name of the family")
		if err := addFamilyCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse arguments: %v", err)
		}

		if *familyName == "" {
			log.Fatal("Family name is required")
		}

		var family models.Family
		if err := database.DB.Where("name = ?", *familyName).First(&family).Error; err == nil {
			fmt.Printf("Family '%s' already exists (ID: %d)\n", family.Name, family.ID)
		} else {
			family = models.Family{Name: *familyName}
			if err := database.DB.Create(&family).Error; err != nil {
				log.Fatalf("Failed to create family: %v", err)
			}
			fmt.Printf("Family '%s' created (ID: %d)\n", family.Name, family.ID)
		}

	case "add-user":
		addUserCmd := flag.NewFlagSet("add-user", flag.ExitOnError)
		userFamily := addUserCmd.String("family", "", "Family name")
		username := addUserCmd.String("username", "", "Username")
		password := addUserCmd.String("password", "", "Password")
		if err := addUserCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse arguments: %v", err)
		}

		if *userFamily == "" || *username == "" || *password == "" {
			log.Fatal("Family, username, and password are required")
		}

		var family models.Family
		if err := database.DB.Where("name = ?", *userFamily).First(&family).Error; err != nil {
			log.Fatalf("Family '%s' not found", *userFamily)
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("Failed to hash password")
		}

		var user models.User
		if err := database.DB.Where("username = ?", *username).First(&user).Error; err == nil {
			// Update existing user
			user.PasswordHash = string(hash)
			user.FamilyID = family.ID
			if err := database.DB.Save(&user).Error; err != nil {
				log.Fatalf("Failed to update user: %v", err)
			}
			fmt.Printf("User '%s' updated in family '%s'\n", user.Username, family.Name)
		} else {
			// Create new user
			user = models.User{
				Username:     *username,
				PasswordHash: string(hash),
				FamilyID:     family.ID,
			}
			if err := database.DB.Create(&user).Error; err != nil {
				log.Fatalf("Failed to create user: %v", err)
			}
			fmt.Printf("User '%s' added to family '%s'\n", user.Username, family.Name)
		}

	case "seed-categories":
		if len(os.Args) < 3 {
			log.Fatal("Family name is required")
		}
		familyName := os.Args[2]

		var family models.Family
		if err := database.DB.Where("name = ?", familyName).First(&family).Error; err != nil {
			log.Fatalf("Family '%s' not found", familyName)
		}

		categories := []models.Category{
			{Name: "Fruits & Vegetables", Icon: "apple", SortOrder: 1},
			{Name: "Bakery", Icon: "croissant", SortOrder: 2},
			{Name: "Dairy & Eggs", Icon: "milk", SortOrder: 3},
			{Name: "Meat & Fish", Icon: "beef", SortOrder: 4},
			{Name: "Beverages", Icon: "cup-soda", SortOrder: 5},
			{Name: "Snacks", Icon: "cookie", SortOrder: 6},
			{Name: "Household", Icon: "home", SortOrder: 7},
			{Name: "Other", Icon: "package", SortOrder: 8},
		}

		for _, cat := range categories {
			cat.FamilyID = family.ID
			database.DB.FirstOrCreate(&cat, models.Category{Name: cat.Name, FamilyID: family.ID})
		}
		fmt.Printf("Seeded %d categories for family '%s'\n", len(categories), family.Name)

	default:
		fmt.Println("expected 'add-family', 'add-user', or 'seed-categories' subcommands")
		os.Exit(1)
	}
}
