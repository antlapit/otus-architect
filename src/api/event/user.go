package event

const (
	TOPIC_USERS           = "user.events"
	EVENT_USER_CREATED    = "user.created"
	EVENT_ADMIN_CREATED   = "admin.created"
	EVENT_CHANGE_PASSWORD = "user.changepassword"
	EVENT_PROFILE_CHANGED = "user.profilechanged"
)

var UserEvents = map[string]interface{}{
	EVENT_USER_CREATED:    UserCreated{},
	EVENT_ADMIN_CREATED:   AdminCreated{},
	EVENT_CHANGE_PASSWORD: UserChangePassword{},
	EVENT_PROFILE_CHANGED: UserProfileChanged{},
}

type UserCreated struct {
	UserId   int64  `json:"userId" binding:"required"`
	Username string `json:"userName" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AdminCreated struct {
	UserId   int64  `json:"userId" binding:"required"`
	Username string `json:"userName" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserChangePassword struct {
	UserId      int64  `json:"userId" binding:"required"`
	Username    string `json:"userName" binding:"required"`
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

type UserProfileChanged struct {
	UserId    int64  `json:"userId" binding:"required"`
	FirstName string `json:"firstName" binding:"required"`
	LastName  string `json:"lastName" binding:"required"`
	Email     string `json:"email" binding:"required"`
	Phone     string `json:"phone" binding:"required"`
}
