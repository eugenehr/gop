package model

// User struct holds user information.
//
// User хранит информацию о пользователе системы.
type User struct {
	Username string       `yaml:"username"`
	Password HashedString `yaml:"password"`
	Groups   []string     `yaml:"groups"`
}

// VerifyPassword verifies user password with given.
//
// VerifyPassword сверяет пользовательский пароль с переданным параметром.
func (u *User) VerifyPassword(password string) bool {
	return u.Password.VerifyPlain(password)
}

// HasAnyGroup returns true if user is member of given groups.
//
// HasAnyGroup возвращает true, если пользователь является членом одной из списка групп.
func (u *User) HasAnyGroup(groups ...string) bool {
	if len(u.Groups) == 0 || len(groups) == 0 {
		return false
	}
	for _, group := range groups {
		for _, ugroup := range u.Groups {
			if ugroup == group {
				return true
			}
		}
	}
	return false
}
