package dep

func Wibble(a, b int) int {
	return a + b
}

func Bar(a, b int) int {
	return a + b
}

type Foo struct {
	a int
}

func NewFoo(a int) *Foo {
	return &Foo{a}
}

func (f *Foo) Wibble(b int) int {
	return f.a * b
}
