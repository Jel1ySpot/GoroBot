package GoroBot

type Service interface {
	Name() string
	Init(*Instant) error
	Release(*Instant) error
}
