package gotiktoklive

type UserNotFound struct{}

func (u UserNotFound) Error() string {
	return "User not found"
}
