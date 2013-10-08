package lib

type Foo struct{}

func NewFoo() *Foo {
	return &Foo{}
}

func (f *Foo) EXPECT() string {
	return "Not Mocked!"
}
