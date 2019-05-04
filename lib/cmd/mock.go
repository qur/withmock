package cmd

type Mock struct {
}

func (m *Mock) Execute(args []string) error {
	return nil
}