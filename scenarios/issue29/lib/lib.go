package lib

type Adder interface {
    Add(x, y int) int
}

type Foo struct {}

func (c *Foo) Add(x, y int) int {
    return x+y
}
