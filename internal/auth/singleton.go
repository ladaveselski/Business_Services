package auth

import (
	"business_services/internal/app/repository"
)

// FixedUserID — константа текущего пользователя.
// По требованию лабы — создатель зафиксирован во всех методах.
const FixedUserID = 1

// CurrentUser возвращает «текущего пользователя» (singleton).
// В 4-й лабе эта функция будет читать ID из JWT/session.
func CurrentUser() repository.User {
	return repository.User{
		ID:    FixedUserID,
		Name:  "creator",
		Email: "creator@example.com",
	}
}

// CurrentUserID — сокращённый доступ, если нужен только ID.
func CurrentUserID() int {
	return FixedUserID
}
