package generate

// go:generate repogen
//go:generate go run ./repogen/

//repogen:entity
type User struct {
	ID           uint `gorm:"primary_key"`
	Email        string
	PasswordHash string
}
