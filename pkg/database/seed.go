package database

import (
	"github.com/Payphone-Digital/gateway/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// DefaultAdmin defines the default admin user credentials
type DefaultAdmin struct {
	FirstName string
	LastName  string
	Email     string
	Password  string
	Phone     string
}

// GetDefaultAdmin returns the default admin user
func GetDefaultAdmin() DefaultAdmin {
	return DefaultAdmin{
		FirstName: "Admin",
		LastName:  "Gateway",
		Email:     "admin@gateway.local",
		Password:  "Admin@123", // Change this in production!
		Phone:     "+6281234567890",
	}
}

// Seed creates initial data for the database
func Seed(db *gorm.DB) error {
	return SeedUsers(db)
}

// SeedUsers creates the default admin user if not exists
func SeedUsers(db *gorm.DB) error {
	admin := GetDefaultAdmin()

	// Check if admin user already exists
	var existingUser model.User
	result := db.Where("email = ?", admin.Email).First(&existingUser)

	if result.Error == nil {
		// User already exists, skip seeding
		return nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		// Unexpected error
		return result.Error
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(admin.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Create the admin user
	user := model.User{
		FirstName:    admin.FirstName,
		LastName:     admin.LastName,
		Email:        admin.Email,
		Password:     string(hashedPassword),
		Phone:        admin.Phone,
		TokenVersion: 1,
	}

	return db.Create(&user).Error
}
