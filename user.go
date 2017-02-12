package main

import (
	"github.com/martini-contrib/sessionauth"
)

type MyUserModel struct {
	Id            int64  `gorm:"primary_key" form:"id" db:"id"`
	Username      string `form:"name" db:"username"`
	Password      string `form:"password" db:"password"`
	authenticated bool   `form:"-" db:"-"`
}

func GenerateAnonymousUser() sessionauth.User {
	return &MyUserModel{}
}

func (u *MyUserModel) Login() {
	u.authenticated = true
}

func (u *MyUserModel) Logout() {
	u.authenticated = false
}

func (u *MyUserModel) IsAuthenticated() bool {
	return u.authenticated
}

func (u *MyUserModel) UniqueId() interface{} {
	return u.Id
}

func (u *MyUserModel) GetById(id interface{}) error {
	err := dbmap.SelectOne(u, "SELECT * FROM users WHERE id = $1", id)
	if err != nil {
		return err
	}

	return nil
}
