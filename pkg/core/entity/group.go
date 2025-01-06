package entity

type Group struct {
	*Base
	Members []*User
}
