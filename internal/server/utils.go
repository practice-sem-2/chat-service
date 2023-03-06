package server

import (
	"github.com/practice-sem-2/user-service/internal/models"
	"github.com/practice-sem-2/user-service/internal/pb"
)

func ParseCreateRequest(req *pb.CreateUserRequest) (models.UserCreate, error) {
	u := models.UserCreate{
		Username:  req.Username,
		Password:  req.Password,
		Email:     req.Email,
		FirstName: "",
		LastName:  "",
		AvatarID:  req.AvatarId,
	}

	if req.FirstName != nil {
		u.FirstName = *req.FirstName
	}

	if req.LastName != nil {
		u.LastName = *req.LastName
	}
	err := models.Validate.Struct(u)
	return u, err
}

func ToUserData(user *models.User) *pb.UserData {
	data := pb.UserData{
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		AvatarId:     user.AvatarID,
		FirstName:    nil,
		LastName:     nil,
	}

	if user.FirstName != "" {
		data.FirstName = &user.FirstName
	}

	if user.LastName != "" {
		data.LastName = &user.LastName
	}
	return &data
}
