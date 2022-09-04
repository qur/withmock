package wmqe_package_name

const wmqe_package = "wmqe_package_name"

var wmqe_main_controller wmqe_controller = wmqe_stub{}

type wmqe_stub struct{}

func (s wmqe_stub) MethodCalled(_ interface{}, _, _, _ string, _ ...interface{}) (bool, []interface{}) {
	return false, nil
}

type wmqe_controller interface {
	MethodCalled(r interface{}, p, t, m string, arguments ...interface{}) (mock bool, ret []interface{})
}

func WMQE_SetController(c wmqe_controller) {
	wmqe_main_controller = c
}
