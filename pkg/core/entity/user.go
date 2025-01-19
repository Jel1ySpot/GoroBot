package entity

type Sender struct {
	*User
	From *Base
}

type User struct {
	*Base
	Nickname  string
	Age       uint32
	Authority Authority
}

type Authority int

const (
	Banned Authority = iota
	Member
	GroupAdmin
	GroupOwner
	Admin
	Owner
)
