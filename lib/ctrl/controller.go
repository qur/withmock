package ctrl

import (
	"sync"

	"github.com/stretchr/testify/mock"
)

type Controller struct {
	lock  sync.Mutex
	mocks map[string]map[any]*mock.Mock
}

var defaultController = newController()

func newController() *Controller {
	return &Controller{
		mocks: make(map[string]map[any]*mock.Mock),
	}
}

func DefaultController() *Controller {
	return defaultController
}

func (c *Controller) createMock(r any, p, t string) *mock.Mock {
	if m := c.findMock(r, p, t); m != nil {
		return m
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	mk := &mock.Mock{}
	key := p + "." + t
	tmk := c.mocks[key]
	if tmk == nil {
		tmk = make(map[any]*mock.Mock)
	}
	tmk[r] = mk
	return mk
}

func (c *Controller) findMock(r any, p, t string) *mock.Mock {
	c.lock.Lock()
	defer c.lock.Unlock()
	key := p + "." + t
	tmk := c.mocks[key]
	if mk, ok := tmk[r]; ok {
		return mk
	}
	return tmk[mock.Anything]
}

func (c *Controller) On(r any, p, t, m string, args ...any) *mock.Call {
	return c.createMock(r, p, t).On(m, args...)
}

func (c *Controller) MethodCalled(r any, p, t, m string, args ...any) (bool, []any) {
	if mk := c.findMock(r, p, t); mk != nil {
		return true, mk.MethodCalled(m, args...)
	}
	return false, nil
}
