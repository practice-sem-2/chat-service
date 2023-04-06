package usecases

import "github.com/google/uuid"

func ValidateUUID(rawUUID string) bool {
	_, err := uuid.Parse(rawUUID)
	return err == nil
}
