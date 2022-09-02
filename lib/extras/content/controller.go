package wmqe_package_name

const wmqe_package = "wmqe_package_name"

var wmqe_main_controller wmqe_controller = wmqe_stub{}

type wmqe_stub struct{}

func (s wmqe_stub) MethodCalled(_, _, _ string, _ ...interface{}) (bool, []interface{}) {
	return false, nil
}

type wmqe_controller interface {
	MethodCalled(p, t, m string, arguments ...interface{}) (mock bool, ret []interface{})
}

func WMQE_SetController(c wmqe_controller) {
	wmqe_main_controller = c
}

type WMQE_Mock struct {
	WMQE_Controller wmqe_controller
}

func (s WMQE_Mock) methodCalled(p, t, m string, arguments ...interface{}) (mock bool, ret []interface{}) {
	if s.WMQE_Controller != nil {
		return s.WMQE_Controller.MethodCalled(p, t, m, arguments...)
	}
	return wmqe_main_controller.MethodCalled(p, t, m, arguments)
}
