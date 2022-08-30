package codemod

type Modifier struct{}

func NewModifier() *Modifier {
	return &Modifier{}
}

func (m *Modifier) Modify(path string) error {
	return nil
}
