package lib

type Adder interface {
    Add(x, y int) int
}

type Splitter interface {
	Split(a int) (x, y int)
}

type Foo struct {}

func (c *Foo) Add(x, y int) int {
    return x+y
}

type Bar struct {}

func (b *Bar) Split(a int) (x, y int) {
	return a-1, 1
}
